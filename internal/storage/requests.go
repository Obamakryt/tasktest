package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
)

func (s *StructPool) Register(ctx context.Context, password, username string) (string, error) {
	const query = `INSERT INTO users (username, pass_hash)
					VALUES ($1, $2) 
					ON CONFLICT DO NOTHING
					 RETURNING username`
	var login string

	err := s.Pool.QueryRow(ctx, query, username, password).Scan(&login)
	if err != nil {
		return "", err
	}
	return login, nil
}
func (s *StructPool) Login(ctx context.Context, password, username string, token string, ttl time.Duration) error {
	const query1 = `SELECT id FROM users 
				WHERE username = $1 AND pass_hash = $2`
	var id int
	err := s.Pool.QueryRow(ctx, query1, username, password).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Invaliddata
		}
		return SomeWrong
	}

	t := Token{Token: token, TimeCreated: time.Now(), TimeExpired: time.Now().Add(ttl)}
	const query2 = `INSERT INTO sessions (token, created_at, expire_at, user_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET token = EXCLUDED.token,
		    created_at = EXCLUDED.created_at,
		    expire_at = EXCLUDED.expire_at`

	commandtag, err := s.Pool.Exec(ctx, query2, t.Token, t.TimeCreated, t.TimeExpired, id)
	if err != nil {
		return SomeWrong
	}
	if commandtag.RowsAffected() == 0 {
		return SomeWrong
	}
	return nil
}
func (s *StructPool) ValidateToken(ctx context.Context, token string) (int, error) {
	const query = `SELECT expire_at, token_id, user_id From sessions WHERE token = $1 `
	data := struct {
		ExpireAt time.Time
		id       int
		userid   int
	}{}

	err := s.Pool.QueryRow(ctx, query, token).Scan(&data.ExpireAt, &data.id, &data.userid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, Invalidtoken
		}
		return 0, SomeWrong
	}

	if time.Since(data.ExpireAt).Seconds() > 0 {
		go func() {
			deletecontext, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			_ = s.DeleteToken(deletecontext, data.id)
		}()
		return 0, Invalidtoken
	}
	return data.userid, nil

}
func (s *StructPool) DeleteToken(ctx context.Context, idUser int) error {
	const query = `DELETE FROM sessions WHERE user_id = $1`
	_, err := s.Pool.Exec(ctx, query, idUser)
	if err != nil {
		return SomeWrong
	}
	return nil
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////
func (s *StructPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return s.Pool.BeginTx(ctx, pgx.TxOptions{})
}

func (s *StructPool) NewDocs(ctx context.Context, dock Dock, tx pgx.Tx) (bool, error) {
	const query = `INSERT INTO documents 
    (id, name, public,is_file,mime,json_data,file_path, own_id)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := tx.Exec(ctx, query, dock.Id, dock.Name, dock.Public,
		dock.IsFile, dock.Mime, dock.Json, dock.Filepath, dock.OwnerId)
	if err != nil {
		return false, err
	}
	return true, nil

}
func (s *StructPool) AddGrant(ctx context.Context, grants []string, docid uuid.UUID, tx pgx.Tx) (bool, error) {
	values := []interface{}{}
	placeholders := []string{}
	argIdx := 1

	for _, grant := range grants {
		var userID int
		err := tx.QueryRow(ctx, "SELECT id FROM users WHERE username=$1", grant).Scan(&userID)
		if err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return false, err
		}
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d)", argIdx, argIdx+1))
		values = append(values, docid, userID)
		argIdx += 2
	}
	query := fmt.Sprintf(`
        INSERT INTO document_grants(document_id, granted_user_id) 
        VALUES %s 
        ON CONFLICT DO NOTHING`, strings.Join(placeholders, ","))

	_, err := tx.Exec(ctx, query, values...)
	if err != nil {
		return false, err
	}
	return true, nil
}
func (s *StructPool) GetDock(ctx context.Context, filter GetDock) ([]DocumentWithGrants, error) {
	allowedColumns := map[string]bool{
		"name":       true,
		"mime":       true,
		"is_file":    true,
		"public":     true,
		"created_at": true,
	}
	if !allowedColumns[filter.Key] {
		return []DocumentWithGrants{}, fmt.Errorf("invalid filter key")
	}

	var (
		rows  pgx.Rows
		err   error
		query string
	)

	if filter.Login == "" {
		query = fmt.Sprintf(`
        SELECT 
            d.id,
            d.name,
            d.mime,
            d.is_file,
            d.public,
            d.created_at,
            d.json_data,
            COALESCE(d.file_path, '') as file_path,
			COALESCE(array_agg(u.username) FILTER (WHERE u.username IS NOT NULL), '{}') as granted_users
        FROM documents d
        LEFT JOIN document_grants g ON d.id = g.document_id
        LEFT JOIN users u ON g.granted_user_id = u.id
        WHERE d.own_id = $1 AND d.%s = $2
        GROUP BY d.id
        LIMIT $3
    `, filter.Key)

		rows, err = s.Pool.Query(ctx, query, filter.Id, filter.Value, filter.Limit)
		if err != nil {
			return nil, err
		}
	} else {
		query = fmt.Sprintf(`
        SELECT 
            d.id,
            d.name,
            d.mime,
            d.is_file,
            d.public,
            d.created_at,
            d.json_data,
            COALESCE(d.file_path, '') as file_path,
			COALESCE(array_agg(u.username) FILTER (WHERE u.username IS NOT NULL), '{}') as granted_users
        FROM documents d
        JOIN users u ON d.own_id = u.id
        LEFT JOIN document_grants g ON d.id = g.document_id
        LEFT JOIN users u2 ON g.granted_user_id = u2.id
        WHERE 
            (
                (d.public = TRUE AND d.own_id = u.id)
                OR
                (g.granted_user_id = u.id)
            )
            AND d.%s = $1
            AND u.username = $2
        GROUP BY d.id
        LIMIT $3
    `, filter.Key)

		rows, err = s.Pool.Query(ctx, query, filter.Value, filter.Login, filter.Limit)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()
	var results []DocumentWithGrants
	for rows.Next() {
		var doc DocumentWithGrants
		if err := rows.Scan(&doc.ID, &doc.Name, &doc.Mime, &doc.IsFile, &doc.Public, &doc.CreatedAt, &doc.Json, &doc.Filepath, &doc.GrantedUsers); err != nil {
			return nil, err
		}
		results = append(results, doc)
	}

	return results, nil
}

func (s *StructPool) GetDockById(ctx context.Context, idUser int, idDock uuid.UUID) (DocumentWithGrants, error) {
	data := DocumentWithGrants{}
	const query = `SELECT d.id,
            d.name,
            d.mime,
            d.is_file,
            d.public,
            d.created_at,
            d.json_data,
        		COALESCE(d.file_path, '') as file_path,
			COALESCE(array_agg(u.username) FILTER (WHERE u.username IS NOT NULL), '{}') as granted_users
FROM documents d 
LEFT JOIN document_grants g ON d.id = g.document_id
LEFT JOIN users u ON g.granted_user_id = u.id
WHERE d.id = $1 and d.own_id = $2
GROUP BY d.id
`

	err := s.Pool.QueryRow(ctx, query, idDock, idUser).Scan(&data.ID, &data.Name,
		&data.Mime, &data.IsFile, &data.Public, &data.CreatedAt, &data.Json, &data.Filepath, &data.GrantedUsers)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DocumentWithGrants{}, Invaliddata
		}
		return DocumentWithGrants{}, err
	}
	return data, nil
}

func (s *StructPool) DeleteDock(ctx context.Context, idUser int, idDock uuid.UUID) error {
	const query = `DELETE FROM documents WHERE id = $1 AND own_id = $2`

	commandtag, err := s.Pool.Exec(ctx, query, idDock, idUser)
	if err != nil {
		return Internal
	}
	if commandtag.RowsAffected() == 0 {
		return Forbidden
	}
	return nil
}

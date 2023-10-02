package postgresql

import (
	// "context"
	"database/sql"
	"errors"
	"fmt"
	"url_shortener/internal/config"

	"url_shortener/internal/storage"

	"github.com/lib/pq" // для загрузки драйверов БД
	// "github.com/jackc/pgconn"
	// "github.com/jackc/pgx/v4"
)

type Storage struct {
	db *sql.DB
}

// type Client interface {
// 	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
// 	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
// 	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
// }

func New(sc config.StorageConfig) (*Storage, error) {
	const op = "storage.postgresql.New" // это просто адрес, где лежит функция New()

	// connection details
	psqlconn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", sc.Host, sc.Port, sc.Username, sc.Password, sc.Database)

	// open existing DB
	db, err := sql.Open("postgres", psqlconn) // postgres - имя драйвера
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// establish connection to DB
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	// create DB table
	stmt, err := db.Prepare(`
    CREATE TABLE IF NOT EXISTS url(
        id SERIAL PRIMARY KEY,
        alias TEXT NOT NULL UNIQUE,
        url TEXT NOT NULL);
    `)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// create index
	stmt, err = db.Prepare(`
    CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
    `)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveURL(urlToSave, alias string) error {
	const op = "storage.postgresql.SaveURL"

	// Подготавливаем запрос
	stmt, err := s.db.Prepare(`INSERT INTO url(alias, url) VALUES($1, $2)`)
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	defer stmt.Close()

	// Выполняем запрос
	_, err = stmt.Exec(alias, urlToSave)
	if err != nil {
		// необходимо добавить проверку на то, что данный url уже есть в базе: storage.ErrURLExists
		// подробнее тут: https://pkg.go.dev/github.com/lib/pq#Error
		if err, ok := err.(*pq.Error); ok {
			fmt.Println("pq error:", err.Code.Name())
		}
		return fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.postgresql.GetURL"

	stmt, err := s.db.Prepare(`SELECT url FROM url WHERE alias = $1`)
	if err != nil {
		return "", fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	defer stmt.Close()

	var resURL string

	err = stmt.QueryRow(alias).Scan(&resURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrURLNotFound
	}
	if err != nil {
		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return resURL, nil
}

package metadb

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	typesv1 "github.com/aybabtme/syncy/pkg/gen/types/v1"
	_ "github.com/go-sql-driver/mysql"
	"github.com/noquark/nanoid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Metadata interface {
	CreateAccount(ctx context.Context, accountName string) (accountPublicID string, err error)
	CreateProject(ctx context.Context, accountPublicID, projectName string) error
	Stat(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path) (*typesv1.FileInfo, bool, error)
	ListDir(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error)
	GetSignature(ctx context.Context, accountPublicID, projectName string) (*typesv1.DirSum, error)
	CreatePathTx(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, fi *typesv1.FileInfo, fn FileSaveAction) error
	PatchPathTx(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn FileSaveAction) error
}

type FileSaveAction func(filepath string) (blake3_64_256_sum []byte, err error)

var _ Metadata = (*MySQL)(nil)

type MySQL struct {
	db *sql.DB
}

func NewMySQL(db *sql.DB) *MySQL {
	return &MySQL{db: db}
}

func (ms *MySQL) CreateAccount(ctx context.Context, accountName string) (accountPublicID string, err error) {
	accountPublicID = nanoid.Must(12)
	_, err = ms.db.ExecContext(ctx, "INSERT INTO accounts (`name`, `public_id`) VALUES (?, ?)",
		accountName, accountPublicID,
	)
	if err != nil {
		return "", fmt.Errorf("inserting account: %w", err)
	}
	return accountPublicID, nil
}

func (ms *MySQL) CreateProject(ctx context.Context, accountPublicID, projectName string) error {
	accountInternalID, ok, err := findAccountID(ctx, ms.db, accountPublicID)
	if err != nil {
		return fmt.Errorf("looking up account: %w", err)
	}
	if !ok {
		return fmt.Errorf("account doesn't exist, create it first")
	}

	_, err = ms.db.ExecContext(ctx, "INSERT INTO projects (`account_id`, `name`) VALUES (?, ?)",
		accountInternalID, projectName,
	)
	if err != nil {
		return fmt.Errorf("inserting project: %w", err)
	}
	return nil
}

func (ms *MySQL) Stat(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path) (*typesv1.FileInfo, bool, error) {
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectName)
	if err != nil || !ok {
		return nil, false, err
	}
	if len(path.Elements) == 0 {
		return nil, false, fmt.Errorf("can't stat a dir, use `listDir` instead")
	}
	var parentDirID *uint64
	if len(path.Elements) > 1 {
		n := len(path.Elements) - 1
		parentDir := &typesv1.Path{Elements: path.Elements[:n]}
		parentDirID, ok, err = findDirID(ctx, ms.db, projectID, parentDir)
		if err != nil || !ok {
			return nil, false, err
		}
	}
	filename := path.Elements[len(path.Elements)-1]
	return getFileInfo(ctx, ms.db, projectID, parentDirID, filename)
}

func (ms *MySQL) ListDir(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path) ([]*typesv1.FileInfo, bool, error) {
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectName)
	if err != nil || !ok {
		return nil, false, err
	}
	var dirID *uint64
	if len(path.Elements) > 1 {
		dirID, ok, err = findDirID(ctx, ms.db, projectID, path)
		if err != nil || !ok {
			return nil, false, err
		}
	}

	dirfis, err := listDirs(ctx, ms.db, projectID, dirID)
	if err != nil {
		return nil, false, err
	}
	filefis, err := listFiles(ctx, ms.db, projectID, dirID)
	if err != nil {
		return nil, false, err
	}
	all := append(dirfis, filefis...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})
	return all, true, nil // TODO: add pagination
}

func listDirs(ctx context.Context, querier querier, projectID uint64, dirID *uint64) ([]*typesv1.FileInfo, error) {
	var (
		out  []*typesv1.FileInfo
		rows *sql.Rows
		err  error
	)
	// TODO: CTE to get dir size
	if dirID != nil {
		rows, err = querier.QueryContext(ctx,
			"SELECT "+dirInfoColumns+" FROM dirs\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `parent_id` = ?\n"+
				"LIMIT 10000", // TODO: pagination
			projectID, *dirID,
		)
	} else {
		rows, err = querier.QueryContext(ctx,
			"SELECT "+dirInfoColumns+" FROM dirs\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `parent_id` IS NULL AND\n"+
				"      `name` = ?\n"+
				"LIMIT 10000", // TODO: pagination
			projectID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("querying for dirs in dir: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		fi, _, err := scanDirInfo(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		out = append(out, fi)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}
	return out, nil
}

func listFiles(ctx context.Context, querier querier, projectID uint64, dirID *uint64) ([]*typesv1.FileInfo, error) {
	var (
		out  []*typesv1.FileInfo
		rows *sql.Rows
		err  error
	)
	if dirID != nil {
		rows, err = querier.QueryContext(ctx,
			"SELECT "+fileInfoColumns+" FROM files\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `dir_id` = ?\n"+
				"LIMIT 10000", // TODO: pagination
			projectID, *dirID,
		)
	} else {
		rows, err = querier.QueryContext(ctx,
			"SELECT "+fileInfoColumns+" FROM files\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `dir_id` IS NULL AND\n"+
				"      `name` = ?\n"+
				"LIMIT 10000", // TODO: pagination
			projectID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("querying for files in dir: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		fi, _, err := scanFileInfo(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		out = append(out, fi)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}
	return out, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func getFileID(ctx context.Context, querier querier, projectID uint64, parentDirID *uint64, filename string) (uint64, bool, error) {
	var row *sql.Row
	if parentDirID != nil {
		row = querier.QueryRowContext(ctx,
			"SELECT `id` FROM files\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `dir_id` = ? AND\n"+
				"      `name` = ?\n"+
				"LIMIT 1",
			projectID, *parentDirID, filename,
		)
	} else {
		row = querier.QueryRowContext(ctx,
			"SELECT `id` FROM files\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `dir_id` IS NULL AND\n"+
				"      `name` = ?\n"+
				"LIMIT 1",
			projectID, filename,
		)
	}
	var id uint64
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	return id, true, nil
}

func getFileInfo(ctx context.Context, querier querier, projectID uint64, parentDirID *uint64, filename string) (*typesv1.FileInfo, bool, error) {
	var row *sql.Row
	if parentDirID != nil {
		row = querier.QueryRowContext(ctx,
			"SELECT "+fileInfoColumns+" FROM files\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `dir_id` = ? AND\n"+
				"      `name` = ?\n"+
				"LIMIT 1",
			projectID, *parentDirID, filename,
		)
	} else {
		row = querier.QueryRowContext(ctx,
			"SELECT "+fileInfoColumns+" FROM files\n"+
				"WHERE `project_id` = ? AND\n"+
				"      `dir_id` IS NULL AND\n"+
				"      `name` = ?\n"+
				"LIMIT 1",
			projectID, filename,
		)
	}
	return scanFileInfo(row)
}

const dirInfoColumns = "`name`, `mod_time`, `mode`"

func scanDirInfo(row scanner) (*typesv1.FileInfo, bool, error) {
	var modTimeUnix int64
	fi := new(typesv1.FileInfo)
	if err := row.Scan(
		&fi.Name,
		&modTimeUnix,
		&fi.Mode,
	); err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	fi.IsDir = true
	fi.ModTime = timestamppb.New(time.Unix(modTimeUnix, 0))
	return fi, true, nil
}

const fileInfoColumns = "`name`, `size`, `mod_time`, `mode`"

func scanFileInfo(row scanner) (*typesv1.FileInfo, bool, error) {
	var modTimeUnix int64
	fi := new(typesv1.FileInfo)
	if err := row.Scan(
		&fi.Name,
		&fi.Size,
		&modTimeUnix,
		&fi.Mode,
	); err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	fi.IsDir = false
	fi.ModTime = timestamppb.New(time.Unix(modTimeUnix, 0))
	return fi, true, nil
}

func (ms *MySQL) GetSignature(ctx context.Context, accountPublicID, projectName string) (*typesv1.DirSum, error) {
	panic("todo")
}

func filepathName(accountPublicID, projectName string, path *typesv1.Path) string {
	return filepath.Join(accountPublicID, projectName, filepath.Join(path.Elements...))
}

func (ms *MySQL) CreatePathTx(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, fi *typesv1.FileInfo, fn FileSaveAction) error {
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectName)
	if err != nil {
		return fmt.Errorf("finding project ID: %w", err)
	}
	if !ok {
		return fmt.Errorf("project doesn't exist, create it first")
	}

	filepath := filepathName(accountPublicID, projectName, path)

	var pendingFileID uint64
	err = withTx(ctx, ms.db, func(tx *sql.Tx) error {
		var parentDirID *uint64
		if len(path.Elements) > 0 {
			id, ok, err := findDirID(ctx, tx, projectID, path)
			if err != nil {
				return fmt.Errorf("finding dir %q, %w", typesv1.StringFromPath(path), err)
			}
			if !ok {
				return fmt.Errorf("parent dir %q doesn't exist, you must create it first", typesv1.StringFromPath(path))
			}
			parentDirID = id
		}
		if fi.IsDir {
			_, err := fn(filepath)
			if err != nil {
				return err
			}
			_, err = createDir(ctx, tx, projectID, parentDirID, fi.Name, fi)
			return err
		}

		pendingFileID, err = createPendingFile(ctx, tx, projectID, parentDirID, fi.Name, fi)
		if err != nil {
			return fmt.Errorf("creating pending file entry: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if fi.IsDir {
		return nil // we're done
	}
	sum, err := fn(filepath)
	if err != nil {
		return err
	}

	return finishPendingFile(ctx, ms.db, pendingFileID, sum)
}

func (ms *MySQL) PatchPathTx(ctx context.Context, accountPublicID, projectName string, path *typesv1.Path, fi *typesv1.FileInfo, sum *typesv1.FileSum, fn FileSaveAction) error {
	projectID, ok, err := findProjectID(ctx, ms.db, accountPublicID, projectName)
	if err != nil {
		return fmt.Errorf("finding project ID: %w", err)
	}
	if !ok {
		return fmt.Errorf("project doesn't exist, create it first")
	}

	filepath := filepathName(accountPublicID, projectName, path)

	var pendingFileID uint64
	err = withTx(ctx, ms.db, func(tx *sql.Tx) error {
		var parentDirID *uint64
		if len(path.Elements) > 0 {
			id, ok, err := findDirID(ctx, tx, projectID, path)
			if err != nil {
				return fmt.Errorf("finding dir %q, %w", typesv1.StringFromPath(path), err)
			}
			if !ok {
				return fmt.Errorf("parent dir %q doesn't exist, you must create it first", typesv1.StringFromPath(path))
			}
			parentDirID = id
		}
		if fi.IsDir {
			// we just update metadata so there's no need to mark anything pending
			_, err := fn(filepath)
			if err != nil {
				return err
			}
			return updateDirInfo(ctx, tx, projectID, parentDirID, fi.Name, fi)
		}

		fileID, ok, err := getFileID(ctx, tx, projectID, parentDirID, fi.Name)
		if err != nil {
			return fmt.Errorf("looking up file: %w", err)
		}
		if !ok {
			return fmt.Errorf("file doesn't exist, cannot be patched")
		}

		pendingFileID, err = markFileAsPending(ctx, tx, projectID, fileID, fi)
		if err != nil {
			return fmt.Errorf("creating pending file entry: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if fi.IsDir {
		return nil // we're done
	}
	blake3sum, err := fn(filepath)
	if err != nil {
		return err
	}

	return finishPendingFile(ctx, ms.db, pendingFileID, blake3sum)
}

func createPendingFile(ctx context.Context, execer execer, projectID uint64, parentDirID *uint64, name string, fi *typesv1.FileInfo) (uint64, error) {
	var (
		res sql.Result
		err error
	)
	if parentDirID != nil {
		res, err = execer.ExecContext(ctx,
			"INSERT INTO files (`project_id`, `dir_id`, `name`, `size`, `mod_time`, `mode`) VALUES (?,?,?,?,?,?)",
			projectID,
			parentDirID,
			name,
			fi.ModTime.AsTime().Unix(),
			fi.Mode,
		)
	} else {
		res, err = execer.ExecContext(ctx,
			"INSERT INTO files (`project_id`, `name`, `size`, `mod_time`, `mode`) VALUES (?,?,?,?,?)",
			projectID,
			parentDirID,
			name,
			fi.ModTime.AsTime().Unix(),
			fi.Mode,
		)
	}
	if err != nil {
		return 0, fmt.Errorf("inserting file: %w", err)
	}
	fileID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting file ID: %w", err)
	}
	res, err = execer.ExecContext(ctx,
		"INSERT INTO pending_files (file_id) VALUES (?)",
		fileID,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting pending file: %w", err)
	}
	_ = res
	return uint64(fileID), nil
}

func markFileAsPending(ctx context.Context, execer execer, projectID uint64, fileID uint64, fi *typesv1.FileInfo) (uint64, error) {
	res, err := execer.ExecContext(ctx,
		"INSERT INTO pending_files (`file_id`) VALUES (?)",
		fileID,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting pending file: %w", err)
	}
	_ = res
	return uint64(fileID), nil
}

func finishPendingFile(ctx context.Context, db *sql.DB, pendingFileID uint64, blake3_64_256_sum []byte) error {
	return withTx(ctx, db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE files SET blake3_64_256_sum=? WHERE file_id = ? LIMIT 1", pendingFileID)
		if err != nil {
			return fmt.Errorf("updating file with blake3_64_256_sum: %w", err)
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM pending_files WHERE file_id = ? LIMIT 1", pendingFileID)
		if err != nil {
			return fmt.Errorf("deleting pending file entry: %w", err)
		}
		return nil
	})
}

func finishPendingPatchFile(ctx context.Context, db *sql.DB, pendingFileID uint64, fi *typesv1.FileInfo, blake3_64_256_sum []byte) error {
	return withTx(ctx, db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			"UPDATE files\n"+
				"SET\n"+
				"	`size` = ?,\n"+
				"	`mod_time` = ?,\n"+
				"	`mode` = ?,\n"+
				"	`blake3_64_256_sum` = ?\n"+
				"WHERE file_id = ? LIMIT 1",
			fi.Size,
			fi.ModTime.AsTime().Unix(),
			fi.Mode,
			pendingFileID,
		)
		if err != nil {
			return fmt.Errorf("updating file with blake3_64_256_sum: %w", err)
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM pending_files WHERE file_id = ? LIMIT 1", pendingFileID)
		if err != nil {
			return fmt.Errorf("deleting pending file entry: %w", err)
		}
		return nil
	})
}

func updateDirInfo(ctx context.Context, execer execer, projectID uint64, parentDirID *uint64, name string, fi *typesv1.FileInfo) error {
	var err error
	if parentDirID != nil {
		_, err = execer.ExecContext(ctx,
			"UPDATE dirs\n"+
				"SET\n"+
				"	name=?,\n"+
				"	mod_time=?,\n"+
				"	mode=?\n"+
				"WHERE `project_id` = ? AND\n"+
				"	`parent_id` = ? AND\n"+
				"	`name` = ?\n"+
				"LIMIT 1",
			projectID, *parentDirID, name,
		)
	} else {
		_, err = execer.ExecContext(ctx,
			"UPDATE dirs\n"+
				"SET\n"+
				"	name=?,\n"+
				"	mod_time=?,\n"+
				"	mode=?\n"+
				"WHERE `project_id` = ? AND\n"+
				"	`parent_id` IS NULL AND\n"+
				"	`name` = ?\n"+
				"LIMIT 1",
			projectID, name,
		)
	}
	return err
}

func findAccountID(ctx context.Context, querier querier, accountPublicID string) (uint64, bool, error) {
	var accountID uint64
	err := querier.QueryRowContext(ctx,
		"SELECT accounts.`id` FROM accounts\n"+
			"WHERE accounts.`public_id` = ?\n"+
			"LIMIT 1",
		accountPublicID,
	).Scan(&accountID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	return accountID, true, nil
}

func findProjectID(ctx context.Context, querier querier, accountPublicID, projectName string) (uint64, bool, error) {
	var projectID uint64
	err := querier.QueryRowContext(ctx,
		"SELECT projects.`id` FROM projects\n"+
			"JOIN accounts ON (accounts.`id` = projects.`account_id`)\n"+
			"WHERE accounts.`public_id` = ? AND\n"+
			"      projects.`name` = ?\n"+
			"LIMIT 1",
		accountPublicID,
		projectName,
	).Scan(&projectID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	return projectID, true, err
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func createDir(ctx context.Context,
	execer execer,
	projectID uint64,
	parentDirID *uint64,
	name string,
	fi *typesv1.FileInfo,
) (uint64, error) {
	var (
		res sql.Result
		err error
	)
	if parentDirID != nil {
		res, err = execer.ExecContext(ctx,
			"INSERT INTO dirs (`project_id`, `parent_id`, `name`, `mod_time`, `mode`) VALUES (?, ?, ?, ?, ?)",
			projectID,
			parentDirID,
			name,
			fi.ModTime.AsTime().Unix(),
			fi.Mode,
		)
	} else {
		res, err = execer.ExecContext(ctx,
			"INSERT INTO dirs (`project_id`, `name`, `mod_time`, `mode`) VALUES (?, ?, ?, ?)",
			projectID,
			name,
			fi.ModTime.AsTime().Unix(),
			fi.Mode,
		)
	}
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func findDirID(
	ctx context.Context,
	querier querier,
	projectID uint64,
	path *typesv1.Path,
) (*uint64, bool, error) {
	if len(path.Elements) == 0 {
		return nil, true, nil
	}
	var dirID uint64
	err := querier.QueryRowContext(ctx,
		"SELECT id FROM dirs WHERE `project_id` = ? AND `parent_id` IS NULL AND `name` = ? LIMIT 1",
		projectID,
		path.Elements[0],
	).Scan(&dirID)
	if err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("finding first element of path: %w", err)
	}
	if len(path.Elements) == 1 {
		return &dirID, true, nil
	}
	for _, elem := range path.Elements[1:] {
		err = querier.QueryRowContext(ctx,
			"SELECT id FROM dirs WHERE `project_id` = ? AND `parent_id` = ? AND `name` = ? LIMIT 1",
			projectID,
			dirID,
			elem,
		).Scan(&dirID)
		if err == sql.ErrNoRows {
			return nil, false, nil
		} else if err != nil {
			return nil, false, fmt.Errorf("finding %q: %w", strings.Join(path.Elements, "/"), err)
		}
	}
	return &dirID, true, nil
}

func withTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commiting transaction: %w", err)
	}
	return nil
}

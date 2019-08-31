package data

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
)

var db *sql.DB

// 启动数据库
func openDB(driver, source string) (err error) {
	db, err = sql.Open(driver, source)
	if err != nil {
		return fmt.Errorf("打开数据库失败: %v", err)
	}

	err = initDB()
	if err != nil {
		return fmt.Errorf("初始化数据库失败: %v", err)
	}

	return nil
}

// 关闭数据库
func closeDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// 初始化数据库
func initDB() error {
	// "QQ->UUID", "UUID->QQ", "QQ->Level",
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users(
		    QQ INTEGER PRIMARY KEY ,
		    UUID BLOB NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS auths(
		    QQ INTEGER PRIMARY KEY ,
		    Level INT DEFAULT 0
		);
	`)
	if err != nil {
		return err
	}
	return nil
}

// SetWhitelist 尝试向数据库写入白名单数据，当ID未被占用时返回自己的QQ，当ID被占用则返回占用者的QQ
// 若原本该账号占有一个UUID，则会返回当时的UUID
func SetWhitelist(QQ int64, ID uuid.UUID, onOldID func(oldID uuid.UUID) error, onSuccess func() error) (owner int64, err error) {
	var tx *sql.Tx
	tx, err = db.Begin()
	if err != nil {
		err = fmt.Errorf("数据库开始事务失败: %v", err)
		return
	}

	// 在函数结束时根据err判断是否应该Rollback或者Commit
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = fmt.Errorf("数据库操作失败: %v，且无法回滚数据: %v", err, rollbackErr)
			}
		} else {
			if err = tx.Commit(); err != nil {
				err = fmt.Errorf("数据提交失败: %v", err)
			}
		}
	}()

	var rows *sql.Rows
	// 检查UUID是否被他人占用
	if rows, err = tx.Query("SELECT QQ FROM users WHERE UUID=?", ID[:]); err != nil {
		err = fmt.Errorf("数据库查询是否有占用者失败: %v", err)
		return
	}
	// 返回占用该账号的人
	if rows.Next() {
		if err = rows.Scan(&owner); err != nil {
			err = fmt.Errorf("数据库读取占用者失败: %v", err)
			return
		}
		if owner != QQ {
			return // 有人占用这这个账号，返回占用者qq和nil
		}
	}
	owner = QQ //没人占用的话所有者就是自己

	// 查询是否有旧白名单
	if rows, err = tx.Query("SELECT UUID FROM users WHERE QQ=?", QQ); err != nil {
		err = fmt.Errorf("查询旧UUID失败: %v", err)
		return
	}

	if rows.Next() {
		// 消除旧账号白名单
		var oldID uuid.UUID
		if err = rows.Scan(&oldID); err != nil {
			err = fmt.Errorf("数据库读取旧UUID失败: %v", err)
			return
		}

		if err = onOldID(oldID); err != nil {
			return
		}

		// 更新UUID
		if _, err = tx.Exec("UPDATE users SET UUID=? WHERE QQ=?", ID[:], QQ); err != nil {
			err = fmt.Errorf("数据库更新UUID失败: %v", err)
			return
		}
	} else {
		// 插入UUID
		if _, err = tx.Exec("INSERT INTO users (QQ, UUID) VALUES (?,?)", QQ, ID[:]); err != nil {
			err = fmt.Errorf("数据库插入UUID失败: %v", err)
			return
		}
	}

	// 更新玩家UUID
	if err = onSuccess(); err != nil {
		return
	}
	return
}

// UnsetWhitelist 从数据库获取玩家绑定的ID，返回UUID并删除记录
func UnsetWhitelist(QQ int64, onHas func(ID uuid.UUID) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("数据库开始事务失败: %v", err)
	}

	// 在函数结束时根据err判断是否应该Rollback或者Commit
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = fmt.Errorf("数据库操作失败: %v，且无法回滚数据: %v", err, rollbackErr)
			}
		} else {
			if err = tx.Commit(); err != nil {
				err = fmt.Errorf("数据提交失败: %v", err)
			}
		}
	}()

	rows, err := tx.Query("SELECT UUID FROM users WHERE QQ=?", QQ)
	if err != nil {
		return fmt.Errorf("数据库查询UUID失败: %v", err)
	}

	if rows.Next() {
		var oldID uuid.UUID
		if err := rows.Scan(&oldID); err != nil {
			return fmt.Errorf("数据库读取旧UUID失败: %v", err)
		}

		if err := onHas(oldID); err != nil {
			return err
		}

		if _, err := tx.Exec("DELETE FROM users WHERE QQ=?", QQ); err != nil {
			return fmt.Errorf("数据库删除UUID失败: %v", err)
		}
		fmt.Println("成功删除数据")
	}
	fmt.Println("成功设置")
	return nil
}

// GetLevel 获取某人的权限等级
func GetLevel(QQ int64) (level int64, err error) {
	var tx *sql.Tx
	tx, err = db.Begin()
	defer func() {
		rbErr := tx.Rollback()
		if rbErr != nil {
			err = rbErr
		}
	}()

	var rows *sql.Rows
	rows, err = tx.Query("SELECT Level FROM auths WHERE QQ=?", QQ)
	if err != nil {
		return
	}

	if rows.Next() {
		err = rows.Scan(&level)
		return
	}
	level = 0
	return
}

// SetLevel 设置某人的权限等级
func SetLevel(QQ, level int64) (err error) {
	var tx *sql.Tx
	tx, err = db.Begin()

	// 查询是否有记录
	var rows *sql.Rows
	rows, err = tx.Query("SELECT Level FROM auths WHERE QQ=?", QQ)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("数据库操作失败: %v，且无法回滚数据: %v", err, rollbackErr)
		}
		return fmt.Errorf("数据库查询等级失败: %v", err)
	}

	// 根据数据存在性判断采用INSERT还是UPDATE
	if rows.Next() {
		_, err = tx.Exec("UPDATE auths SET Level=? WHERE QQ=?", level, QQ)
	} else {
		_, err = tx.Exec("INSERT INTO auths (QQ, Level) VALUES (?,?)", QQ, level)
	}
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("数据库操作失败: %v，且无法回滚数据: %v", err, rollbackErr)
		}
		return fmt.Errorf("数据库操作失败: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("数据库提交数据失败: %v", err)
	}
	return
}

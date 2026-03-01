package test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"seata.apache.org/seata-go/v2/pkg/client"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

func init() {
	// 设置配置文件路径
	absPath, _ := filepath.Abs("../testdata/conf/seatago.yml")
	os.Setenv("SEATA_GO_CONFIG_PATH", absPath)
}

// TestXAIssue904 复现 issue #904
// XA 模式下 SELECT FOR UPDATE 后 UPDATE 报 busy buffer / bad connection
func TestXAIssue904(t *testing.T) {
	// MySQL 连接信息
	const mysqlDSN = "root:1234@tcp(127.0.0.1:3306)/seata_test?charset=utf8mb4"

	t.Log("=== 测试 issue #904: XA 模式 SELECT FOR UPDATE 后 UPDATE ===")

	// 初始化 Seata 客户端
	client.Init()
	t.Log("Seata client initialized")

	// 创建 XA 数据源
	db, err := sql.Open("seata-xa-mysql", mysqlDSN)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	t.Log("XA 数据源创建成功")

	// 重置测试数据
	resetTestData(t, mysqlDSN)

	// 执行 XA 事务测试
	t.Run("SELECT_FOR_UPDATE_then_UPDATE", func(t *testing.T) {
		err = tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
			Name:    "test-xa-issue904",
			Timeout: 60 * time.Second,
		}, func(ctx context.Context) error {
			// 第一步：查询 (带 FOR UPDATE)
			var money int
			err := db.QueryRowContext(ctx,
				"SELECT money FROM account_tbl WHERE user_id = ? FOR UPDATE", "user1").
				Scan(&money)
			if err != nil {
				return fmt.Errorf("query failed: %w", err)
			}
			t.Logf("Step 1: 查询成功, money = %d", money)

			// 第二步：更新
			newBalance := money - 10
			_, err = db.ExecContext(ctx,
				"UPDATE account_tbl SET money = ? WHERE user_id = ?", newBalance, "user1")
			if err != nil {
				return fmt.Errorf("update failed: %w", err)
			}
			t.Logf("Step 2: 更新成功, new balance = %d", newBalance)
			return nil
		})

		if err != nil {
			t.Errorf("❌ 事务失败: %v", err)
		} else {
			t.Log("✅ 事务成功")
		}
	})
}

func resetTestData(t *testing.T, dsn string) {
	nativeDB, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to open native db: %v", err)
	}
	defer nativeDB.Close()

	// 创建表
	_, err = nativeDB.Exec(`
		CREATE TABLE IF NOT EXISTS account_tbl (
			id int(11) NOT NULL AUTO_INCREMENT,
			user_id varchar(255) DEFAULT NULL,
			money int(11) DEFAULT 0,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8
	`)
	if err != nil {
		t.Logf("Warning: create table failed: %v", err)
	}

	// 重置数据
	nativeDB.Exec("DELETE FROM account_tbl WHERE user_id = 'user1'")
	_, err = nativeDB.Exec("INSERT INTO account_tbl (user_id, money) VALUES ('user1', 100)")
	if err != nil {
		t.Logf("Warning: insert data failed: %v", err)
	}

	t.Log("测试数据已重置")
}

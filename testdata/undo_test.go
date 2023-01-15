package testdata

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/client"
	sql2 "github.com/seata/seata-go/pkg/datasource/sql"
	"github.com/seata/seata-go/pkg/datasource/sql/undo/mysql"
)

func init() {
	client.InitPath("./conf/seatago.yml")
}

func TestUndo(t *testing.T) {
	testUndoLog := func() {
		manager := mysql.NewUndoLogManager()

		db, err := sql.Open(sql2.SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_client?multiStatements=true")
		assert.Nil(t, err)

		ctx := context.Background()
		sqlConn, err := db.Conn(ctx)
		assert.Nil(t, err)

		defer func() {
			_ = sqlConn.Close()
		}()

		if err = manager.RunUndo(ctx, "36375516866494489", 36375516866494490, db, "seata_client"); err != nil {
			t.Logf("%+v", err)
		}

		assert.Nil(t, err)
	}

	t.Run("test_undo_log", func(t *testing.T) {
		testUndoLog()
	})
}

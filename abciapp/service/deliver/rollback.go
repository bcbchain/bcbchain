package deliver

import (
	"github.com/bcbchain/bcbchain/common/statedbhelper"
	"os"
)

func (app *AppDeliver) rollback() error {
	app.logger.Info("ROLLBACK")

	home := os.Getenv("HOME")

	if err := os.RemoveAll(home + "/.build/bin"); err != nil {
		return err
	}

	statedbhelper.RollbackStateDB(1)
	return nil
}

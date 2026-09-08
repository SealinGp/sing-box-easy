package initstate

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"xorm.io/xorm"
)

func testEngine() *xorm.Engine {
	e, err := database.GetEngine()
	if err != nil {
		panic(err)
	}
	return e
}

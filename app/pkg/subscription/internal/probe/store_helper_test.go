package subprobe

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/repo"
)

func testStore() *StoreXORM {
	e, err := database.GetEngine()
	if err != nil {
		panic(err)
	}
	return repo.NewProbeStore(e)
}

type StoreXORM = repo.ProbeStore

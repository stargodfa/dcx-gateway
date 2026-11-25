package bootstrap

import (
	"dcx-gateway/internal/cache"
	"dcx-gateway/internal/config"
	"dcx-gateway/internal/repository"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Dependencies struct {
	Config *config.Config
	Logger *zap.Logger
	DB     *gorm.DB
	Redis  *redis.Client
}

type Caches struct {
	Script       cache.ScriptCache
	Module       cache.ModuleCache
	Video        cache.VideoCache
	Auth         cache.AuthCache
	ChartRuntime cache.ChartRuntimeCache
}

type Repositories struct {
	Script           repository.ScriptRepository
	ScriptModule     repository.ScriptModuleRepository
	DataBinding      repository.DataBindingRepository
	VideoSource      repository.VideoSourceRepository
	User             repository.UserRepository
	UEConfigDocument repository.UEConfigDocumentRepository
	UEChartConfig    repository.UEChartConfigRepository
	UESceneDesired   repository.UESceneDesiredModelRepository
	UEModelType      repository.UEModelTypeRepository
	UEModelInstance  repository.UEModelInstanceRepository
	UEModelPlacement repository.UEModelInstancePlacementRepository
	UEAuxConfig      repository.UEAuxConfigRepository
}

func NewMySQL(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

func NewRedis(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

func NewRepositories(db *gorm.DB, log *zap.Logger) Repositories {
	return Repositories{
		Script:           repository.NewScriptRepository(db, log),
		ScriptModule:     repository.NewScriptModuleRepository(db, log),
		DataBinding:      repository.NewDataBindingRepository(db, log),
		VideoSource:      repository.NewVideoSourceRepository(db, log),
		User:             repository.NewUserRepository(db, log),
		UEConfigDocument: repository.NewUEConfigDocumentRepository(db, log),
		UEChartConfig:    repository.NewUEChartConfigRepository(db, log),
		UESceneDesired:   repository.NewUESceneDesiredModelRepository(db, log),
		UEModelType:      repository.NewUEModelTypeRepository(db, log),
		UEModelInstance:  repository.NewUEModelInstanceRepository(db, log),
		UEModelPlacement: repository.NewUEModelInstancePlacementRepository(db, log),
		UEAuxConfig:      repository.NewUEAuxConfigRepository(db, log),
	}
}

func NewCaches(rdb *redis.Client, keyPrefix ...string) Caches {
	prefix := ""
	if len(keyPrefix) > 0 {
		prefix = keyPrefix[0]
	}
	return Caches{
		Script:       cache.NewScriptCache(rdb, prefix),
		Module:       cache.NewModuleCache(rdb, prefix),
		Video:        cache.NewVideoCache(rdb, prefix),
		Auth:         cache.NewAuthCache(rdb, prefix),
		ChartRuntime: cache.NewChartRuntimeCache(rdb, prefix),
	}
}

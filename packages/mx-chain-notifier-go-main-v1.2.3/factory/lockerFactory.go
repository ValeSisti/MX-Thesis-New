package factory

import (
	"github.com/multiversx/mx-chain-notifier-go/common"
	"github.com/multiversx/mx-chain-notifier-go/config"
	"github.com/multiversx/mx-chain-notifier-go/disabled"
	"github.com/multiversx/mx-chain-notifier-go/redis"
)

// CreateLockService creates lock service component based on config
func CreateLockService(checkDuplicates bool, config config.RedisConfig) (redis.LockService, error) {
	//! -------------------- NEW CODE --------------------
	log.Debug("*** CreateLockService called ***")
	//! ---------------- END OF NEW CODE -----------------			
	if !checkDuplicates {
		//! -------------------- NEW CODE --------------------
		log.Debug("*** BEFORE RETURNING !checkDuplicates***")
		//! ---------------- END OF NEW CODE -----------------				
		return disabled.NewDisabledRedlockWrapper(), nil
	}

	redisClient, err := createRedisClient(config)
		//! -------------------- NEW CODE --------------------
		log.Debug("*** BEFORE RETURNING createRedisClient no error***")
		//! ---------------- END OF NEW CODE -----------------			
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("*** BEFORE RETURNING createRedisClient ***")
		//! ---------------- END OF NEW CODE -----------------				
		return nil, err
	}

	redlockArgs := redis.ArgsRedlockWrapper{
		Client:       redisClient,
		TTLInMinutes: config.TTL,
	}
	lockService, err := redis.NewRedlockWrapper(redlockArgs)
	//! -------------------- NEW CODE --------------------
	log.Debug("*** BEFORE RETURNING createRedisClient no error***")
	//! ---------------- END OF NEW CODE -----------------			
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("*** BEFORE RETURNING NewRedlockWrapper ***")
		//! ---------------- END OF NEW CODE -----------------				
		return nil, err
	}
	//! -------------------- NEW CODE --------------------
	log.Debug("*** BEFORE RETURNING ***")
	//! ---------------- END OF NEW CODE -----------------		
	return lockService, nil
}

func createRedisClient(cfg config.RedisConfig) (redis.RedLockClient, error) {
	//! -------------------- NEW CODE --------------------
	log.Debug("*** createRedisClient ***")
	//! ---------------- END OF NEW CODE -----------------		
	switch cfg.ConnectionType {
	case common.RedisInstanceConnType:
		return redis.CreateSimpleClient(cfg)
	case common.RedisSentinelConnType:
		return redis.CreateFailoverClient(cfg)
	default:
		return nil, common.ErrInvalidRedisConnType
	}
}

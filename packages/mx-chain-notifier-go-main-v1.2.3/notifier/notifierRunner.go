package notifier

import (
	"os"
	"os/signal"

	marshalFactory "github.com/multiversx/mx-chain-core-go/marshal/factory"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/multiversx/mx-chain-notifier-go/api/shared"
	"github.com/multiversx/mx-chain-notifier-go/config"
	"github.com/multiversx/mx-chain-notifier-go/facade"
	"github.com/multiversx/mx-chain-notifier-go/factory"
	"github.com/multiversx/mx-chain-notifier-go/metrics"
	"github.com/multiversx/mx-chain-notifier-go/process"
	"github.com/multiversx/mx-chain-notifier-go/rabbitmq"
)

var log = logger.GetOrCreate("notifierRunner")

type notifierRunner struct {
	configs config.Configs
}

// NewNotifierRunner create a new notifierRunner instance
func NewNotifierRunner(cfgs *config.Configs) (*notifierRunner, error) {
	if cfgs == nil {
		return nil, ErrNilConfigs
	}

	return &notifierRunner{
		configs: *cfgs,
	}, nil
}

// Start will trigger the notifier service
func (nr *notifierRunner) Start() error {
	//! -------------------- NEW CODE --------------------
	log.Debug("***Start called ***")
	//! ---------------- END OF NEW CODE -----------------		
	publisherType := nr.configs.Flags.PublisherType
	//! -------------------- NEW CODE --------------------
	log.Debug("*** PRINTING ***", "publisherType", publisherType)
	//! ---------------- END OF NEW CODE -----------------		

	externalMarshaller, err := marshalFactory.NewMarshalizer(nr.configs.MainConfig.General.ExternalMarshaller.Type)
	//! -------------------- NEW CODE --------------------
	log.Debug("***is error nil?? NM ***", "err == nil???", err == nil)
	//! ---------------- END OF NEW CODE -----------------		
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("***Error: during NewMarshalizer ***", "err", err.Error())
		//! ---------------- END OF NEW CODE -----------------			
		return err
	}

	//! -------------------- NEW CODE --------------------
	log.Debug("*** BEFORE CALLS ***", "err == nil???", err == nil)
	//! ---------------- END OF NEW CODE -----------------		

	lockService, err := factory.CreateLockService(nr.configs.MainConfig.General.CheckDuplicates, nr.configs.MainConfig.Redis)
	//! -------------------- NEW CODE --------------------
	log.Debug("***is error nil?? CL ***", "err == nil???", err == nil)
	//! ---------------- END OF NEW CODE -----------------		
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("***Error: during CreateLockService ***", "err", err.Error())
		//! ---------------- END OF NEW CODE -----------------			
		return err
	}

	commonHub, err := factory.CreateHub(publisherType)
	//! -------------------- NEW CODE --------------------
	log.Debug("***is error nil?? CH ***", "err == nil???", err == nil)
	//! ---------------- END OF NEW CODE -----------------		
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("***Error: during CreateHub ***", "err", err.Error())
		//! ---------------- END OF NEW CODE -----------------				
		return err
	}

	publisher, err := factory.CreatePublisher(publisherType, nr.configs.MainConfig, externalMarshaller, commonHub)
	//! -------------------- NEW CODE --------------------
	log.Debug("***is error nil?? CP ***", "err == nil???", err == nil)
	//! ---------------- END OF NEW CODE -----------------	
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("***Error: during CreatePublisher ***", "err", err.Error())
		//! ---------------- END OF NEW CODE -----------------				
		return err
	}
	wsHandler, err := factory.CreateWSHandler(publisherType, commonHub, externalMarshaller)
	//! -------------------- NEW CODE --------------------
	log.Debug("***is error nil?? CWSH ***", "err == nil???", err == nil)
	//! ---------------- END OF NEW CODE -----------------		
	if err != nil {
		//! -------------------- NEW CODE --------------------
		log.Debug("***Error: during CreateWSHandler ***", "err", err.Error())
		//! ---------------- END OF NEW CODE -----------------			
		return err
	}

	statusMetricsHandler := metrics.NewStatusMetrics()

	eventsInterceptor, err := factory.CreateEventsInterceptor(nr.configs.MainConfig.General)
	if err != nil {
		return err
	}

	argsEventsHandler := process.ArgsEventsHandler{
		CheckDuplicates:      nr.configs.MainConfig.General.CheckDuplicates,
		Locker:               lockService,
		Publisher:            publisher,
		StatusMetricsHandler: statusMetricsHandler,
		EventsInterceptor:    eventsInterceptor,
	}
	eventsHandler, err := process.NewEventsHandler(argsEventsHandler)
	if err != nil {
		return err
	}

	facadeArgs := facade.ArgsNotifierFacade{
		EventsHandler:        eventsHandler,
		APIConfig:            nr.configs.MainConfig.ConnectorApi,
		WSHandler:            wsHandler,
		StatusMetricsHandler: statusMetricsHandler,
	}
	facade, err := facade.NewNotifierFacade(facadeArgs)
	if err != nil {
		return err
	}

	webServer, err := factory.CreateWebServerHandler(facade, nr.configs)
	if err != nil {
		return err
	}

	wsConnector, err := factory.CreateWSObserverConnector(nr.configs.MainConfig.WebSocketConnector, facade)
	if err != nil {
		return err
	}

	err = publisher.Run()
	if err != nil {
		return err
	}

	err = webServer.Run()
	if err != nil {
		return err
	}

	err = waitForGracefulShutdown(webServer, publisher, wsConnector)
	if err != nil {
		return err
	}
	log.Debug("closing eventNotifier proxy...")

	return nil
}

func waitForGracefulShutdown(
	server shared.WebServerHandler,
	publisher rabbitmq.PublisherService,
	wsConnector process.WSClient,
) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit

	err := server.Close()
	if err != nil {
		return err
	}

	err = wsConnector.Close()
	if err != nil {
		return err
	}

	err = publisher.Close()
	if err != nil {
		return err
	}

	return nil
}

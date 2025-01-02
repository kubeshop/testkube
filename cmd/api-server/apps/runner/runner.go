package runner

//type Options struct {
//	ClusterID           string
//	DashboardURI        string
//	DefaultNamespace    string
//	ServiceAccountNames map[string]string
//}
//
//func Start(
//	ctx context.Context,
//	logger *zap.SugaredLogger,
//	eventsEmitter event.Interface,
//	grpcConn *grpc.ClientConn,
//	grpcApiToken string,
//	testWorkflowProcessor testworkflowprocessor.Processor,
//	opts Options,
//) {
//	metrics := metrics.NewMetrics()
//
//	//opts = Options{
//	//	ClusterID: opts.ClusterID,
//	//	DefaultNamespace: cfg.TestkubeNamespace,
//	// //  DefaultServiceAccountName: cfg.JobServiceAccountName,
//	//  DashboardURI: cfg.TestkubeDashboardURI,
//	//}
//	//
//	//opts.ServiceAccountNames = map[string]string{
//	//	opts.DefaultNamespace: cfg.JobServiceAccountName,
//	//}
//	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)
//	proContext := commons.ReadProContext(ctx, commons.MustGetConfig(), grpcClient)
//	//subscriptionChecker, err := checktcl.NewSubscriptionChecker(ctx, proContext, grpcClient, grpcConn)
//	//commons.ExitOnError("Subscription checking", err)
//	//// Pro edition only (tcl protected code)
//	//if cfg.TestkubeExecutionNamespaces != "" {
//	//	err = subscriptionChecker.IsActiveOrgPlanEnterpriseForFeature("execution namespace")
//	//	commons.ExitOnError("Subscription checking", err)
//	//	serviceAccountNames = schedulertcl.GetServiceAccountNamesFromConfig(serviceAccountNames, cfg.TestkubeExecutionNamespaces)
//	//}
//
//	clientset, err := k8sclient.ConnectToK8s()
//	commons.ExitOnError("Creating k8s clientset", err)
//
//	executionWorker := services.CreateExecutionWorker(clientset, commons.MustGetConfig(), opts.ClusterID, opts.ServiceAccountNames, testWorkflowProcessor, map[string]string{
//		testworkflowconfig.FeatureFlagNewExecutions:            fmt.Sprintf("%v", cfg.FeatureNewExecutions),
//		testworkflowconfig.FeatureFlagTestWorkflowCloudStorage: fmt.Sprintf("%v", cfg.FeatureTestWorkflowCloudStorage),
//	})
//
//	testWorkflowResultsRepository := cloudtestworkflow.NewCloudRepository(grpcClient, grpcConn, grpcApiToken)
//	testWorkflowOutputRepository := cloudtestworkflow.NewCloudOutputRepository(grpcClient, grpcConn, grpcApiToken, cfg.StorageSkipVerify)
//
//	configMapConfig := commons.MustGetConfigMapConfig(ctx, cfg.APIServerConfig, opts.DefaultNamespace, cfg.TestkubeAnalyticsEnabled)
//
//	// Build the runner
//	runner := runner2.New(
//		executionWorker,
//		testWorkflowOutputRepository,
//		testWorkflowResultsRepository,
//		configMapConfig,
//		eventsEmitter,
//		metrics,
//		opts.DashboardURI,
//		cfg.StorageSkipVerify,
//	)
//
//	// Recover control
//	func() {
//		var list []testkube.TestWorkflowExecution
//		for {
//			// TODO: it should get running only in the context of current runner (worker.List?)
//			list, err = testWorkflowResultsRepository.GetRunning(ctx)
//			if err != nil {
//				logger.Errorw("failed to fetch running executions to recover", "error", err)
//				<-time.After(time.Second)
//				continue
//			}
//			break
//		}
//
//		for i := range list {
//			if (list[i].RunnerId == "" && len(list[i].Signature) == 0) || (list[i].RunnerId != "" && list[i].RunnerId != proContext.EnvID) {
//				continue
//			}
//
//			// TODO: Should it throw error at all?
//			// TODO: Pass hints (namespace, signature, scheduledAt)
//			go func(e *testkube.TestWorkflowExecution) {
//				err := runner.Monitor(ctx, e.Id)
//				if err != nil {
//					logger.Errorw("failed to monitor execution", "id", e.Id, "error", err)
//				}
//			}(&list[i])
//		}
//	}()
//}

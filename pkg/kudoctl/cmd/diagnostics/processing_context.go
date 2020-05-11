package diagnostics

func attachToRoot(ctx *processingContext) string {
	return diagDir
}

func attachToKudoRoot(ctx *processingContext) string {
	return diagDir + "/" + "kudo"
}

func attachToOperator(ctx *processingContext) string {
	return diagDir + "/" + "operator_" + ctx.operatorName
}

func attachToInstance(ctx *processingContext) string {
	return attachToOperator(ctx) + "/" + "instance_" + ctx.instanceName
}

type processingContext struct {
	podNames            []string
	operatorName        string
	operatorVersionName string
	instanceName        string
}

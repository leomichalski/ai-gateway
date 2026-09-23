package api

// APIHandler acts as a root facade. By embedding the domain-specific handlers, 
// it automatically inherits their methods and satisfies the oapi-codegen StrictServerInterface.
type APIHandler struct {
	*ChatHandler
	// *TemplateHandler
	// *ModelHandler
}

// NewAPIHandler builds the root handler from all the specialized handlers.
func NewAPIHandler(chatHandler *ChatHandler) *APIHandler {
	return &APIHandler{
		ChatHandler: chatHandler,
	}
}

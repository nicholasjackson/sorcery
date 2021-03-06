package handlers

type BaseResponse struct {
	StatusEvent string `json:"status_event"`
}

const (
	GET                 = ".get"
	POST                = ".post"
	PUT                 = ".put"
	DELETE              = ".delete"
	CALLED              = ".called"
	SUCCESS             = ".success"
	STARTED             = ".started"
	PROCESS_REDELIVERY  = ".process_redelivery"
	DELETE_REGISTRATION = ".delete_registration"
	NO_ENDPOINT         = ".no_registered_endpoint"
	HANDLE              = ".handle"
	DISPATCH            = ".dispatch"
	NOT_FOUND           = ".not_found"
	ERROR               = ".server_error"
	INVALID_REQUEST     = ".request.invalid_request"
	BAD_REQUEST         = ".request.bad_request"
	VALID_REQUEST       = ".request.valid"
	INVALID_TOKEN       = ".auth.invalid_token"
	NOT_AUTHORISED      = ".auth.not_authorised"
	TOKEN_OK            = ".auth.token_ok"
	HEALTH_HANDLER      = "event_sauce.health"
	EVENT_HANDLER       = "event_sauce.event"
	REGISTER_HANDLER    = "event_sauce.register"
	EVENT_QUEUE         = "event_sauce.event_queue"
	DEAD_LETTER_QUEUE   = "event_sauce.dead_letter_queue"
	WORKER              = ".worker"
)

package astinfo

const (
	NOUSAGE = iota
	CREATOR
	SERVLET
	PRPC
	INITIATOR
	FILTER
	WEBSOCKET
)

const (
	UrlFilter = "urlfilter"
	Url       = "url"

	Creator   = "creator"
	Initiator = "initiator"
	Websocket = "websocket"
	Filter    = "filter"
	Servlet   = "servlet" //用于定义struct是servlet，所以默认groupName是servlet
	Prpc      = "prpc"    //用于定义struct是prpc，所以默认groupName是prpc
	Security  = "security"
)

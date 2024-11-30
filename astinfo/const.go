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
	Url       = "url" // 定义函数为servlet，默认method为POST

	Creator     = "creator"
	Initiator   = "initiator"
	Websocket   = "websocket"
	FilterConst = "filter"
	Servlet     = "servlet" //用于定义struct是servlet，所以默认groupName是servlet
	Prpc        = "prpc"    //用于定义struct是prpc，所以默认groupName是prpc
	Security    = "security"
	ConstMethod = "method"
)

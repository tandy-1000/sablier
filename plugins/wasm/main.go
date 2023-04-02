package main

import (
	"strings"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"golang.org/x/exp/slices"
)

const clusterName = "sablier"

func main() {
	proxywasm.SetVMContext(&vmContext{})
}

type vmContext struct {
	// Embed the default VM context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultVMContext
}

// Override types.DefaultVMContext.
func (*vmContext) NewPluginContext(contextID uint32) types.PluginContext {
	return &pluginContext{}
}

type pluginContext struct {
	// Embed the default plugin context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultPluginContext
}

// Override types.DefaultPluginContext.
func (*pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	return &httpAuthRandom{contextID: contextID}
}

type httpAuthRandom struct {
	// Embed the default http context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultHttpContext
	contextID uint32
}

// Override types.DefaultHttpContext.
func (ctx *httpAuthRandom) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	hs, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get request headers: %v", err)
		return types.ActionContinue
	}
	for _, h := range hs {
		proxywasm.LogInfof("request header: %s: %s", h[0], h[1])
	}

	headers := [][2]string{
		{":method", "GET"}, {":authority", "localhost:10000"}, {"accept", "*/*"}, {":path", "/api/strategies/dynamic?group=default"},
	}
	if _, err := proxywasm.DispatchHttpCall(clusterName, headers, nil, nil,
		50000, httpCallResponseCallback); err != nil {
		proxywasm.LogCriticalf("dipatch httpcall failed: %v", err)
		return types.ActionContinue
	}

	proxywasm.LogInfof("http call dispatched to %s", clusterName)

	return types.ActionPause
}

func httpCallResponseCallback(numHeaders, bodySize, numTrailers int) {
	hs, err := proxywasm.GetHttpCallResponseHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get response headers: %v", err)
		return
	}

	headerIndex := slices.IndexFunc(hs, func(h [2]string) bool { return strings.ToLower(h[0]) == "x-sablier-session-status" })

	headerValue := hs[headerIndex][1]

	if headerValue != "ready" {
		b, err := proxywasm.GetHttpCallResponseBody(0, bodySize)
		if err != nil {
			proxywasm.LogCriticalf("failed to get response body: %v", err)
			proxywasm.ResumeHttpRequest()
			return
		}
		if err := proxywasm.SendHttpResponse(403, [][2]string{
			{"powered-by", "proxy-wasm-go-sdk!!"},
		}, b, -1); err != nil {
			proxywasm.LogErrorf("failed to send local response: %v", err)
			proxywasm.ResumeHttpRequest()
		}
	} else {
		proxywasm.ResumeHttpRequest()
	}
}

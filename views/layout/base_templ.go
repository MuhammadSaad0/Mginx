// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.747
package layout

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

import "mginx/views/components"

func BaseLayout() templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<!doctype html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><title>MGINX</title><meta name=\"description\" content=\"Simple Reverse Proxy.\"></head><link rel=\"stylesheet\" href=\"/dist/tailwind.css\"><script src=\"https://unpkg.com/htmx.org@2.0.1\" integrity=\"sha384-QWGpdj554B4ETpJJC9z+ZHJcA/i59TyjxEPXiiUgN2WmTyV5OEZWCD6gQhgkdpB/\" crossorigin=\"anonymous\"></script><script src=\"https://unpkg.com/htmx.org/dist/ext/json-enc.js\"></script><script src=\"https://unpkg.com/htmx.org@1.9.12/dist/ext/response-targets.js\"></script><body class=\"overflow-hidden\"><header class=\"text-center py-4\"><h1 class=\"text-2xl font-bold\">MGINX</h1></header><main class=\"grid place-items-center bg-blue-300 w-screen h-screen overflow-hidden\"><div class=\"grid grid-cols-2 gap-4 w-full max-w-screen-lg h-full p-4\"><div class=\"flex flex-col items-center justify-center space-y-4 h-full\"><h2 class=\"text-lg font-bold text-black mb-4\">Upstreams</h2><div class=\"rounded-lg bg-slate-100 h-[60vh] w-[30vw] border border-gray-300 overflow-y-auto\" id=\"upstreams-list\" hx-get=\"/config/upstreams\" hx-trigger=\"load, every 10s\" hx-swap=\"innerHTML\"></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = components.AddUpstream().Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div id=\"errors\"></div></div><div class=\"flex flex-col items-center justify-center space-y-4 h-full\"><div id=\"loadBStrat\" class=\"w-full max-w-xs\" hx-get=\"/config/get-load-balancing-strategy\" hx-swap=\"innerHTML\" hx-trigger=\"load, every 20s\"></div><div id=\"loadBStratSelect\" class=\"w-full max-w-xs\" hx-get=\"/config/all-load-balancing-strategies\" hx-trigger=\"load\" hx-swap=\"innerHTML\"></div></div></div></main></body></html>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

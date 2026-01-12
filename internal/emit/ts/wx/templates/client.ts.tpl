/* AUTO-GENERATED FILE - DO NOT EDIT */

import { rpcRequest, RpcError } from "./transport";
import * as T from "./types.gen";

export function makeApi(baseURL: string, options?: { headers?: Record<string, string> }) {
  const headers = options?.headers;

  return {
  {{- range .Tags }}
    {{ .Name }}: {
    {{- range .Routes }}
      {{ .Name }}: async ({{ .Signature }}): Promise<{{ .ReturnType }}> => {
        const urlPath = {{ .PathExpr }};
        return rpcRequest<{{ .ReturnType }}>(baseURL, "{{ .Method }}", urlPath, {
          query: {{ .QueryVar }},
          body: {{ .BodyVar }},
          headers,
        });
      },
    {{- end }}
    },
  {{- end }}
  } as const;
}

export { RpcError };

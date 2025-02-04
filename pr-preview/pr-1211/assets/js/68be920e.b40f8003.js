"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[7294],{60266:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>a,contentTitle:()=>c,default:()=>l,frontMatter:()=>o,metadata:()=>n,toc:()=>d});const n=JSON.parse('{"id":"architecture/observability","title":"Observability","description":"The Contrast Coordinator can expose metrics in the","source":"@site/versioned_docs/version-1.1/architecture/observability.md","sourceDirName":"architecture","slug":"/architecture/observability","permalink":"/contrast/pr-preview/pr-1211/1.1/architecture/observability","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.1/architecture/observability.md","tags":[],"version":"1.1","frontMatter":{},"sidebar":"docs","previous":{"title":"Security considerations","permalink":"/contrast/pr-preview/pr-1211/1.1/architecture/security-considerations"},"next":{"title":"Planned features and limitations","permalink":"/contrast/pr-preview/pr-1211/1.1/features-limitations"}}');var s=r(74848),i=r(28453);const o={},c="Observability",a={},d=[{value:"Exposed metrics",id:"exposed-metrics",level:2},{value:"Service mesh metrics",id:"service-mesh-metrics",level:2}];function h(e){const t={a:"a",br:"br",code:"code",h1:"h1",h2:"h2",header:"header",p:"p",...(0,i.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(t.header,{children:(0,s.jsx)(t.h1,{id:"observability",children:"Observability"})}),"\n",(0,s.jsxs)(t.p,{children:["The Contrast Coordinator can expose metrics in the\n",(0,s.jsx)(t.a,{href:"https://prometheus.io/",children:"Prometheus"})," format. These can be monitored to quickly\nidentify problems in the gRPC layer or attestation errors. Prometheus metrics\nare numerical values associated with a name and additional key/values pairs,\ncalled labels."]}),"\n",(0,s.jsx)(t.h2,{id:"exposed-metrics",children:"Exposed metrics"}),"\n",(0,s.jsxs)(t.p,{children:["The metrics can be accessed at the Coordinator pod at the port specified in the\n",(0,s.jsx)(t.code,{children:"CONTRAST_METRICS_PORT"})," environment variable under the ",(0,s.jsx)(t.code,{children:"/metrics"})," endpoint. By\ndefault, this environment variable isn't specified, hence no metrics will be\nexposed."]}),"\n",(0,s.jsxs)(t.p,{children:["The Coordinator exports gRPC metrics under the prefix ",(0,s.jsx)(t.code,{children:"contrast_grpc_server_"}),".\nThese metrics are labeled with the gRPC service name and method name.\nMetrics of interest include ",(0,s.jsx)(t.code,{children:"contrast_grpc_server_handled_total"}),", which counts\nthe number of requests by return code, and\n",(0,s.jsx)(t.code,{children:"contrast_grpc_server_handling_seconds_bucket"}),", which produces a histogram of",(0,s.jsx)(t.br,{}),"\n","request latency."]}),"\n",(0,s.jsxs)(t.p,{children:["The gRPC service ",(0,s.jsx)(t.code,{children:"userapi.UserAPI"})," records metrics for the methods\n",(0,s.jsx)(t.code,{children:"SetManifest"})," and ",(0,s.jsx)(t.code,{children:"GetManifest"}),", which get called when ",(0,s.jsx)(t.a,{href:"../deployment#set-the-manifest",children:"setting the\nmanifest"})," and ",(0,s.jsx)(t.a,{href:"../deployment#verify-the-coordinator",children:"verifying the\nCoordinator"})," respectively."]}),"\n",(0,s.jsxs)(t.p,{children:["The ",(0,s.jsx)(t.code,{children:"meshapi.MeshAPI"})," service records metrics for the method ",(0,s.jsx)(t.code,{children:"NewMeshCert"}),", which\ngets called by the ",(0,s.jsx)(t.a,{href:"/contrast/pr-preview/pr-1211/1.1/components/overview#the-initializer",children:"Initializer"})," when starting a\nnew workload. Attestation failures from workloads to the Coordinator can be\ntracked with the counter ",(0,s.jsx)(t.code,{children:"contrast_meshapi_attestation_failures_total"}),"."]}),"\n",(0,s.jsxs)(t.p,{children:["The current manifest generation is exposed as a\n",(0,s.jsx)(t.a,{href:"https://prometheus.io/docs/concepts/metric_types/#gauge",children:"gauge"})," with the metric\nname ",(0,s.jsx)(t.code,{children:"contrast_coordinator_manifest_generation"}),". If no manifest is set at the\nCoordinator, this counter will be zero."]}),"\n",(0,s.jsx)(t.h2,{id:"service-mesh-metrics",children:"Service mesh metrics"}),"\n",(0,s.jsxs)(t.p,{children:["The ",(0,s.jsx)(t.a,{href:"/contrast/pr-preview/pr-1211/1.1/components/service-mesh",children:"Service Mesh"})," can be configured to expose\nmetrics via its ",(0,s.jsx)(t.a,{href:"https://www.envoyproxy.io/docs/envoy/latest/operations/admin",children:"Envoy admin\ninterface"}),". Be\naware that the admin interface can expose private information and allows\ndestructive operations to be performed. To enable the admin interface for the\nService Mesh, set the annotation\n",(0,s.jsx)(t.code,{children:"contrast.edgeless.systems/servicemesh-admin-interface-port"})," in the configuration\nof your workload. If this annotation is set, the admin interface will be started\non this port."]}),"\n",(0,s.jsxs)(t.p,{children:["To access the admin interface, the ingress settings of the Service Mesh have to\nbe configured to allow access to the specified port (see ",(0,s.jsx)(t.a,{href:"../components/service-mesh#configuring-the-proxy",children:"Configuring the\nProxy"}),"). All metrics will be\nexposed under the ",(0,s.jsx)(t.code,{children:"/stats"})," endpoint. Metrics in Prometheus format can be scraped\nfrom the ",(0,s.jsx)(t.code,{children:"/stats/prometheus"})," endpoint."]})]})}function l(e={}){const{wrapper:t}={...(0,i.R)(),...e.components};return t?(0,s.jsx)(t,{...e,children:(0,s.jsx)(h,{...e})}):h(e)}},28453:(e,t,r)=>{r.d(t,{R:()=>o,x:()=>c});var n=r(96540);const s={},i=n.createContext(s);function o(e){const t=n.useContext(i);return n.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function c(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:o(e.components),n.createElement(i.Provider,{value:t},e.children)}}}]);
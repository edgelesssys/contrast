"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[5231],{3554:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>a,contentTitle:()=>o,default:()=>l,frontMatter:()=>i,metadata:()=>c,toc:()=>h});var n=r(4848),s=r(8453);const i={},o="Observability",c={id:"architecture/observability",title:"Observability",description:"The Contrast Coordinator can expose metrics in the",source:"@site/versioned_docs/version-0.7/architecture/observability.md",sourceDirName:"architecture",slug:"/architecture/observability",permalink:"/contrast/pr-preview/pr-828/0.7/architecture/observability",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.7/architecture/observability.md",tags:[],version:"0.7",frontMatter:{},sidebar:"docs",previous:{title:"Certificate authority",permalink:"/contrast/pr-preview/pr-828/0.7/architecture/certificates"},next:{title:"Planned features and limitations",permalink:"/contrast/pr-preview/pr-828/0.7/features-limitations"}},a={},h=[{value:"Exposed metrics",id:"exposed-metrics",level:2},{value:"Service mesh metrics",id:"service-mesh-metrics",level:2}];function d(e){const t={a:"a",code:"code",h1:"h1",h2:"h2",header:"header",p:"p",...(0,s.R)(),...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.header,{children:(0,n.jsx)(t.h1,{id:"observability",children:"Observability"})}),"\n",(0,n.jsxs)(t.p,{children:["The Contrast Coordinator can expose metrics in the\n",(0,n.jsx)(t.a,{href:"https://prometheus.io/",children:"Prometheus"})," format. These can be monitored to quickly\nidentify problems in the gRPC layer or attestation errors. Prometheus metrics\nare numerical values associated with a name and additional key/values pairs,\ncalled labels."]}),"\n",(0,n.jsx)(t.h2,{id:"exposed-metrics",children:"Exposed metrics"}),"\n",(0,n.jsxs)(t.p,{children:["The metrics can be accessed at the Coordinator pod at the port specified in the\n",(0,n.jsx)(t.code,{children:"CONTRAST_METRICS_PORT"})," environment variable under the ",(0,n.jsx)(t.code,{children:"/metrics"})," endpoint. By\ndefault, this environment variable isn't specified, hence no metrics will be\nexposed."]}),"\n",(0,n.jsxs)(t.p,{children:["The Coordinator starts two gRPC servers, one for the user API on port ",(0,n.jsx)(t.code,{children:"1313"})," and\none for the mesh API on port ",(0,n.jsx)(t.code,{children:"7777"}),". Metrics for both servers can be accessed\nusing different prefixes."]}),"\n",(0,n.jsxs)(t.p,{children:["All metric names for the user API are prefixed with ",(0,n.jsx)(t.code,{children:"contrast_userapi_grpc_server_"}),".\nExposed metrics include the number of  handled requests of the methods\n",(0,n.jsx)(t.code,{children:"SetManifest"})," and ",(0,n.jsx)(t.code,{children:"GetManifest"}),", which get called when ",(0,n.jsx)(t.a,{href:"../deployment#set-the-manifest",children:"setting the\nmanifest"})," and ",(0,n.jsx)(t.a,{href:"../deployment#verify-the-coordinator",children:"verifying the\nCoordinator"})," respectively. For each method\nyou can see the gRPC status code indicating whether the request succeeded or\nnot and the request latency."]}),"\n",(0,n.jsxs)(t.p,{children:["For the mesh API, the metric names are prefixed with ",(0,n.jsx)(t.code,{children:"contrast_meshapi_grpc_server_"}),". The\nmetrics include similar data to the user API for the method ",(0,n.jsx)(t.code,{children:"NewMeshCert"})," which\ngets called by the ",(0,n.jsx)(t.a,{href:"../components#the-initializer",children:"Initializer"})," when starting a\nnew workload. Attestation failures from workloads to the Coordinator can be\ntracked with the counter ",(0,n.jsx)(t.code,{children:"contrast_meshapi_attestation_failures"}),"."]}),"\n",(0,n.jsxs)(t.p,{children:["The current manifest generation is exposed as a\n",(0,n.jsx)(t.a,{href:"https://prometheus.io/docs/concepts/metric_types/#gauge",children:"gauge"})," with the metric\nname ",(0,n.jsx)(t.code,{children:"contrast_coordinator_manifest_generation"}),". If no manifest is set at the\nCoordinator, this counter will be zero."]}),"\n",(0,n.jsx)(t.h2,{id:"service-mesh-metrics",children:"Service mesh metrics"}),"\n",(0,n.jsxs)(t.p,{children:["The ",(0,n.jsx)(t.a,{href:"/contrast/pr-preview/pr-828/0.7/components/service-mesh",children:"Service Mesh"})," can be configured to expose\nmetrics via its ",(0,n.jsx)(t.a,{href:"https://www.envoyproxy.io/docs/envoy/latest/operations/admin",children:"Envoy admin\ninterface"}),". Be\naware that the admin interface can expose private information and allows\ndestructive operations to be performed. To enable the admin interface for the\nService Mesh, set the annotation\n",(0,n.jsx)(t.code,{children:"contrast.edgeless.systems/servicemesh-admin-interface-port"})," in the configuration\nof your workload. If this annotation is set, the admin interface will be started\non this port."]}),"\n",(0,n.jsxs)(t.p,{children:["To access the admin interface, the ingress settings of the Service Mesh have to\nbe configured to allow access to the specified port (see ",(0,n.jsx)(t.a,{href:"../components/service-mesh#configuring-the-proxy",children:"Configuring the\nProxy"}),"). All metrics will be\nexposed under the ",(0,n.jsx)(t.code,{children:"/stats"})," endpoint. Metrics in Prometheus format can be scraped\nfrom the ",(0,n.jsx)(t.code,{children:"/stats/prometheus"})," endpoint."]})]})}function l(e={}){const{wrapper:t}={...(0,s.R)(),...e.components};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(d,{...e})}):d(e)}},8453:(e,t,r)=>{r.d(t,{R:()=>o,x:()=>c});var n=r(6540);const s={},i=n.createContext(s);function o(e){const t=n.useContext(i);return n.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function c(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:o(e.components),n.createElement(i.Provider,{value:t},e.children)}}}]);
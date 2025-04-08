"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[7464],{31535:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>a,contentTitle:()=>c,default:()=>l,frontMatter:()=>o,metadata:()=>s,toc:()=>h});const s=JSON.parse('{"id":"components/service-mesh","title":"Service mesh","description":"The Contrast service mesh secures the communication of the workload by automatically","source":"@site/versioned_docs/version-1.3/components/service-mesh.md","sourceDirName":"components","slug":"/components/service-mesh","permalink":"/contrast/pr-preview/pr-1352/1.3/components/service-mesh","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.3/components/service-mesh.md","tags":[],"version":"1.3","frontMatter":{},"sidebar":"docs","previous":{"title":"Policies","permalink":"/contrast/pr-preview/pr-1352/1.3/components/policies"},"next":{"title":"Attestation","permalink":"/contrast/pr-preview/pr-1352/1.3/architecture/attestation"}}');var i=t(74848),r=t(28453);const o={},c="Service mesh",a={},h=[{value:"Configuring the proxy",id:"configuring-the-proxy",level:2},{value:"Ingress",id:"ingress",level:3},{value:"Egress",id:"egress",level:3}];function d(e){const n={a:"a",code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",p:"p",pre:"pre",ul:"ul",...(0,r.R)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(n.header,{children:(0,i.jsx)(n.h1,{id:"service-mesh",children:"Service mesh"})}),"\n",(0,i.jsxs)(n.p,{children:["The Contrast service mesh secures the communication of the workload by automatically\nwrapping the network traffic inside mutual TLS (mTLS) connections. The\nverification of the endpoints in the connection establishment is based on\ncertificates that are part of the\n",(0,i.jsx)(n.a,{href:"/contrast/pr-preview/pr-1352/1.3/architecture/certificates",children:"PKI of the Coordinator"}),"."]}),"\n",(0,i.jsxs)(n.p,{children:["The service mesh can be enabled on a per-workload basis by adding a service mesh\nconfiguration to the workload's object annotations. During the ",(0,i.jsx)(n.code,{children:"contrast generate"}),"\nstep, the service mesh is added as a ",(0,i.jsx)(n.a,{href:"https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/",children:"sidecar\ncontainer"})," to\nall workloads which have a specified configuration. The service mesh container first\nsets up ",(0,i.jsx)(n.code,{children:"iptables"})," rules based on its configuration and then starts\n",(0,i.jsx)(n.a,{href:"https://www.envoyproxy.io/",children:"Envoy"})," for TLS origination and termination."]}),"\n",(0,i.jsx)(n.h2,{id:"configuring-the-proxy",children:"Configuring the proxy"}),"\n",(0,i.jsx)(n.p,{children:"The service mesh container can be configured using the following object annotations:"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.code,{children:"contrast.edgeless.systems/servicemesh-ingress"})," to configure ingress."]}),"\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.code,{children:"contrast.edgeless.systems/servicemesh-egress"})," to configure egress."]}),"\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.code,{children:"contrast.edgeless.systems/servicemesh-admin-interface-port"})," to configure the Envoy\nadmin interface. If not specified, no admin interface will be started."]}),"\n"]}),"\n",(0,i.jsxs)(n.p,{children:["If you aren't using the automatic service mesh injection and want to configure the\nservice mesh manually, set the environment variables ",(0,i.jsx)(n.code,{children:"CONTRAST_INGRESS_PROXY_CONFIG"}),",\n",(0,i.jsx)(n.code,{children:"CONTRAST_EGRESS_PROXY_CONFIG"})," and ",(0,i.jsx)(n.code,{children:"CONTRAST_ADMIN_PORT"})," in the service mesh sidecar directly."]}),"\n",(0,i.jsx)(n.h3,{id:"ingress",children:"Ingress"}),"\n",(0,i.jsxs)(n.p,{children:["All TCP ingress traffic is routed over Envoy by default. Since we use\n",(0,i.jsx)(n.a,{href:"https://docs.kernel.org/networking/tproxy.html",children:"TPROXY"}),", the destination address\nremains the same throughout the packet handling."]}),"\n",(0,i.jsxs)(n.p,{children:["Any incoming connection is required to present a client certificate signed by the\n",(0,i.jsx)(n.a,{href:"/contrast/pr-preview/pr-1352/1.3/architecture/certificates#usage-of-the-different-certificates",children:"mesh CA certificate"}),".\nEnvoy presents a certificate chain of the mesh\ncertificate of the workload and the intermediate CA certificate as the server certificate."]}),"\n",(0,i.jsxs)(n.p,{children:["If the deployment contains workloads which should be reachable from outside the\nService Mesh, while still handing out the certificate chain, disable client\nauthentication by setting the annotation ",(0,i.jsx)(n.code,{children:"contrast.edgeless.systems/servicemesh-ingress"})," as\n",(0,i.jsx)(n.code,{children:"<name>#<port>#false"}),". Separate multiple entries with ",(0,i.jsx)(n.code,{children:"##"}),". You can choose any\ndescriptive string identifying the service on the given port for the ",(0,i.jsx)(n.code,{children:"<name>"})," field,\nas it's only informational."]}),"\n",(0,i.jsxs)(n.p,{children:["Disable redirection and TLS termination altogether by specifying\n",(0,i.jsx)(n.code,{children:"<name>#<port>#true"}),". This can be beneficial if the workload itself handles TLS\non that port or if the information exposed on this port is non-sensitive."]}),"\n",(0,i.jsx)(n.p,{children:"The following example workload exposes a web service on port 8080 and metrics on\nport 7890. The web server is exposed to a 3rd party end-user which wants to\nverify the deployment, therefore it's still required that the server hands out\nit certificate chain signed by the mesh CA certificate. The metrics should be\nexposed via TCP without TLS."}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:'apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: web\n  annotations:\n    contrast.edgeless.systems/servicemesh-ingress: "web#8080#false##metrics#7890#true"\nspec:\n  replicas: 1\n  template:\n    spec:\n      runtimeClassName: contrast-cc\n      containers:\n        - name: web-svc\n          image: ghcr.io/edgelesssys/frontend:v1.2.3@...\n          ports:\n            - containerPort: 8080\n              name: web\n            - containerPort: 7890\n              name: metrics\n'})}),"\n",(0,i.jsxs)(n.p,{children:["When invoking ",(0,i.jsx)(n.code,{children:"contrast generate"}),", the resulting deployment will be injected with the\nContrast service mesh as an init container."]}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:'# ...\n      initContainers:\n        - env:\n            - name: CONTRAST_INGRESS_PROXY_CONFIG\n              value: "web#8080#false##metrics#7890#true"\n          image: "ghcr.io/edgelesssys/contrast/service-mesh-proxy:v1.3.0@sha256:05ee434548450ef160882828eb215c8b37eb7fa3c9ca33b4a57d977189326400"\n          name: contrast-service-mesh\n          restartPolicy: Always\n          securityContext:\n            capabilities:\n              add:\n                - NET_ADMIN\n            privileged: true\n          volumeMounts:\n            - name: contrast-secrets\n              mountPath: /contrast\n'})}),"\n",(0,i.jsxs)(n.p,{children:["Note, that changing the environment variables of the sidecar container directly will\nonly have an effect if the workload isn't configured to automatically generate a\nservice mesh component on ",(0,i.jsx)(n.code,{children:"contrast generate"}),". Otherwise, the service mesh sidecar\ncontainer will be regenerated on every invocation of the command."]}),"\n",(0,i.jsx)(n.h3,{id:"egress",children:"Egress"}),"\n",(0,i.jsx)(n.p,{children:"To be able to route the egress traffic of the workload through Envoy, the remote\nendpoints' IP address and port must be configurable."}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsxs)(n.li,{children:["Choose an IP address inside the ",(0,i.jsx)(n.code,{children:"127.0.0.0/8"})," CIDR and a port not yet in use\nby the pod."]}),"\n",(0,i.jsx)(n.li,{children:"Configure the workload to connect to this IP address and port."}),"\n",(0,i.jsxs)(n.li,{children:["Set ",(0,i.jsx)(n.code,{children:"<name>#<chosen IP>:<chosen port>#<original-hostname-or-ip>:<original-port>"}),"\nas the ",(0,i.jsx)(n.code,{children:"contrast.edgeless.systems/servicemesh-egress"})," workload annotation. Separate multiple\nentries with ",(0,i.jsx)(n.code,{children:"##"}),". Choose any string identifying the service on the given port as\n",(0,i.jsx)(n.code,{children:"<name>"}),"."]}),"\n"]}),"\n",(0,i.jsxs)(n.p,{children:["This redirects the traffic over Envoy. The endpoint must present a valid\ncertificate chain which must be verifiable with the\n",(0,i.jsx)(n.a,{href:"/contrast/pr-preview/pr-1352/1.3/architecture/certificates#usage-of-the-different-certificates",children:"mesh CA certificate"}),".\nFurthermore, Envoy uses a certificate chain with the mesh certificate of the workload\nand the intermediate CA certificate as the client certificate."]}),"\n",(0,i.jsxs)(n.p,{children:["The following example workload has no ingress connections and two egress\nconnection to different microservices. The microservices are part\nof the confidential deployment. One is reachable under ",(0,i.jsx)(n.code,{children:"billing-svc:8080"})," and\nthe other under ",(0,i.jsx)(n.code,{children:"cart-svc:8080"}),"."]}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:'apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: web\n  annotations:\n    contrast.edgeless.systems/servicemesh-egress: "billing#127.137.0.1:8081#billing-svc:8080##cart#127.137.0.2:8081#cart-svc:8080"\nspec:\n  replicas: 1\n  template:\n    spec:\n      runtimeClassName: contrast-cc\n      containers:\n        - name: currency-conversion\n          image: ghcr.io/edgelesssys/conversion:v1.2.3@...\n'})})]})}function l(e={}){const{wrapper:n}={...(0,r.R)(),...e.components};return n?(0,i.jsx)(n,{...e,children:(0,i.jsx)(d,{...e})}):d(e)}},28453:(e,n,t)=>{t.d(n,{R:()=>o,x:()=>c});var s=t(96540);const i={},r=s.createContext(i);function o(e){const n=s.useContext(r);return s.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function c(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:o(e.components),s.createElement(r.Provider,{value:n},e.children)}}}]);
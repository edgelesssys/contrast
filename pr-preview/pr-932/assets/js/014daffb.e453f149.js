"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[1955],{98711:(e,t,s)=>{s.r(t),s.d(t,{assets:()=>c,contentTitle:()=>o,default:()=>l,frontMatter:()=>i,metadata:()=>a,toc:()=>d});var n=s(74848),r=s(28453);const i={},o="Secrets & recovery",a={id:"architecture/secrets",title:"Secrets & recovery",description:"When the Coordinator is configured with the initial manifest, it generates a random secret seed.",source:"@site/docs/architecture/secrets.md",sourceDirName:"architecture",slug:"/architecture/secrets",permalink:"/contrast/pr-preview/pr-932/next/architecture/secrets",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/docs/architecture/secrets.md",tags:[],version:"current",frontMatter:{},sidebar:"docs",previous:{title:"Attestation",permalink:"/contrast/pr-preview/pr-932/next/architecture/attestation"},next:{title:"Certificate authority",permalink:"/contrast/pr-preview/pr-932/next/architecture/certificates"}},c={},d=[{value:"Persistence",id:"persistence",level:2},{value:"Recovery",id:"recovery",level:2},{value:"Workload Secrets",id:"workload-secrets",level:2},{value:"Secure persistence",id:"secure-persistence",level:3}];function h(e){const t={a:"a",admonition:"admonition",code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",p:"p",pre:"pre",...(0,r.R)(),...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.header,{children:(0,n.jsx)(t.h1,{id:"secrets--recovery",children:"Secrets & recovery"})}),"\n",(0,n.jsx)(t.p,{children:"When the Coordinator is configured with the initial manifest, it generates a random secret seed.\nFrom this seed, it uses an HKDF to derive the CA root key and a signing key for the manifest history.\nThis derivation is deterministic, so the seed can be used to bring any Coordinator to this Coordinator's state."}),"\n",(0,n.jsxs)(t.p,{children:["The secret seed is returned to the user on the first call to ",(0,n.jsx)(t.code,{children:"contrast set"}),", encrypted with the user's public seed share owner key.\nIf no seed share owner key is provided, a key is generated and stored in the working directory."]}),"\n",(0,n.jsx)(t.h2,{id:"persistence",children:"Persistence"}),"\n",(0,n.jsxs)(t.p,{children:["The Coordinator runs as a ",(0,n.jsx)(t.code,{children:"StatefulSet"})," with a dynamically provisioned persistent volume.\nThis volume stores the manifest history and the associated runtime policies.\nThe manifest isn't considered sensitive information, because it needs to be passed to the untrusted infrastructure in order to start workloads.\nHowever, the Coordinator must ensure its integrity and that the persisted data corresponds to the manifests set by authorized users.\nThus, the manifest is stored in plain text, but is signed with a private key derived from the Coordinator's secret seed."]}),"\n",(0,n.jsx)(t.h2,{id:"recovery",children:"Recovery"}),"\n",(0,n.jsxs)(t.p,{children:["When a Coordinator starts up, it doesn't have access to the signing secret and can thus not verify the integrity of the persisted manifests.\nIt needs to be provided with the secret seed, from which it can derive the signing key that verifies the signatures.\nThis procedure is called recovery and is initiated by the workload owner.\nThe CLI decrypts the secret seed using the private seed share owner key, verifies the Coordinator and sends the seed through the ",(0,n.jsx)(t.code,{children:"Recover"})," method.\nThe Coordinator recovers its key material and verifies the manifest history signature."]}),"\n",(0,n.jsx)(t.h2,{id:"workload-secrets",children:"Workload Secrets"}),"\n",(0,n.jsxs)(t.p,{children:["The Coordinator provides each workload a secret seed during attestation.\nThis secret can be used by the workload to derive additional secrets for example to encrypt persistent data.\nLike the workload certificates, it's written to the ",(0,n.jsx)(t.code,{children:"secrets/workload-secret-seed"})," path under the shared Kubernetes volume ",(0,n.jsx)(t.code,{children:"contrast-secrets"}),"."]}),"\n",(0,n.jsx)(t.admonition,{type:"warning",children:(0,n.jsx)(t.p,{children:"The workload owner can decrypt data encrypted with secrets derived from the workload secret.\nThe workload owner can derive the workload secret themselves, since it's derived from the secret seed known to the workload owner.\nIf the data owner and the workload owner is the same entity, then they can safely use the workload secrets."})}),"\n",(0,n.jsx)(t.h3,{id:"secure-persistence",children:"Secure persistence"}),"\n",(0,n.jsx)(t.p,{children:"Remember that persistent volumes from the cloud provider are untrusted.\nUsing the workload secret, applications can set up trusted storage on top of untrusted block devices.\nThe following, slightly abbreviated resource outlines how this could be realized:"}),"\n",(0,n.jsx)(t.admonition,{type:"warning",children:(0,n.jsx)(t.p,{children:"This configuration snippet is intended to be educational and needs to be refined and adapted to your production environment.\nUsing it as-is may result in data corruption or data loss."})}),"\n",(0,n.jsx)(t.pre,{children:(0,n.jsx)(t.code,{className:"language-yaml",children:"apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  name: volume-tester\nspec:\n  template:\n    spec:\n      containers:\n      - name: main\n        image: my.registry/my-image@sha256:0123...\n        command:\n        - /bin/sh\n        - -ec\n        - | # <-- Custom script that mounts the encrypted disk and then calls the original application.\n          device=/dev/csi0\n          if ! cryptsetup isLuks $device; then\n            cryptsetup luksFormat $device /contrast/secrets/workload-secret-seed\n            cryptsetup open $device state -d /contrast/secrets/workload-secret-seed\n            mkfs.ext4 /dev/mapper/state\n            cryptsetup close state\n          fi\n          cryptsetup open $device state -d /contrast/secrets/workload-secret-seed\n          /path/to/original/app\n        name: volume-tester\n        volumeDevices:\n        - name: state\n          devicePath: /dev/csi0\n        securityContext:\n          privileged: true # <-- This is necessary for mounting devices.\n      runtimeClassName: contrast-cc\n  volumeClaimTemplates:\n  - apiVersion: v1\n    kind: PersistentVolumeClaim\n    metadata:\n      name: state\n    spec:\n      accessModes:\n      - ReadWriteOnce\n      resources:\n        requests:\n          storage: 1Gi\n      volumeMode: Block # <-- The requested volume needs to be a raw block device.\n"})}),"\n",(0,n.jsx)(t.admonition,{type:"note",children:(0,n.jsxs)(t.p,{children:["This example assumes that you can modify the container image to include a shell and the ",(0,n.jsx)(t.code,{children:"cryptsetup"})," utility.\nAlternatively, you can set up a secure mount from a sidecar container inside an ",(0,n.jsx)(t.code,{children:"emptyDir"})," mount shared with the main container.\nThe Contrast end-to-end tests include ",(0,n.jsx)(t.a,{href:"https://github.com/edgelesssys/contrast/blob/0662a2e/internal/kuberesource/sets.go#L504",children:"an example"})," of this type of mount."]})})]})}function l(e={}){const{wrapper:t}={...(0,r.R)(),...e.components};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(h,{...e})}):h(e)}},28453:(e,t,s)=>{s.d(t,{R:()=>o,x:()=>a});var n=s(96540);const r={},i=n.createContext(r);function o(e){const t=n.useContext(i);return n.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function a(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:o(e.components),n.createElement(i.Provider,{value:t},e.children)}}}]);
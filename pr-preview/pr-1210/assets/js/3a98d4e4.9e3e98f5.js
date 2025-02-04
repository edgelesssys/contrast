"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[8515],{86627:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>a,default:()=>l,frontMatter:()=>o,metadata:()=>s,toc:()=>d});const s=JSON.parse('{"id":"architecture/secrets","title":"Secrets & recovery","description":"When the Coordinator is configured with the initial manifest, it generates a random secret seed.","source":"@site/versioned_docs/version-1.4/architecture/secrets.md","sourceDirName":"architecture","slug":"/architecture/secrets","permalink":"/contrast/pr-preview/pr-1210/architecture/secrets","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.4/architecture/secrets.md","tags":[],"version":"1.4","frontMatter":{},"sidebar":"docs","previous":{"title":"Attestation","permalink":"/contrast/pr-preview/pr-1210/architecture/attestation"},"next":{"title":"Certificate authority","permalink":"/contrast/pr-preview/pr-1210/architecture/certificates"}}');var r=n(74848),i=n(28453);const o={},a="Secrets & recovery",c={},d=[{value:"Persistence",id:"persistence",level:2},{value:"Recovery",id:"recovery",level:2},{value:"Workload Secrets",id:"workload-secrets",level:2},{value:"Secure persistence",id:"secure-persistence",level:3},{value:"Usage <code>cryptsetup</code> subcommand",id:"usage-cryptsetup-subcommand",level:4}];function h(e){const t={admonition:"admonition",code:"code",h1:"h1",h2:"h2",h3:"h3",h4:"h4",header:"header",p:"p",pre:"pre",...(0,i.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(t.header,{children:(0,r.jsx)(t.h1,{id:"secrets--recovery",children:"Secrets & recovery"})}),"\n",(0,r.jsx)(t.p,{children:"When the Coordinator is configured with the initial manifest, it generates a random secret seed.\nFrom this seed, it uses an HKDF to derive the CA root key and a signing key for the manifest history.\nThis derivation is deterministic, so the seed can be used to bring any Coordinator to this Coordinator's state."}),"\n",(0,r.jsxs)(t.p,{children:["The secret seed is returned to the user on the first call to ",(0,r.jsx)(t.code,{children:"contrast set"}),", encrypted with the user's public seed share owner key.\nIf no seed share owner key is provided, a key is generated and stored in the working directory."]}),"\n",(0,r.jsx)(t.h2,{id:"persistence",children:"Persistence"}),"\n",(0,r.jsxs)(t.p,{children:["The Coordinator runs as a ",(0,r.jsx)(t.code,{children:"StatefulSet"})," with a dynamically provisioned persistent volume.\nThis volume stores the manifest history and the associated runtime policies.\nThe manifest isn't considered sensitive information, because it needs to be passed to the untrusted infrastructure in order to start workloads.\nHowever, the Coordinator must ensure its integrity and that the persisted data corresponds to the manifests set by authorized users.\nThus, the manifest is stored in plain text, but is signed with a private key derived from the Coordinator's secret seed."]}),"\n",(0,r.jsx)(t.h2,{id:"recovery",children:"Recovery"}),"\n",(0,r.jsxs)(t.p,{children:["When a Coordinator starts up, it doesn't have access to the signing secret and can thus not verify the integrity of the persisted manifests.\nIt needs to be provided with the secret seed, from which it can derive the signing key that verifies the signatures.\nThis procedure is called recovery and is initiated by the workload owner.\nThe CLI decrypts the secret seed using the private seed share owner key, verifies the Coordinator and sends the seed through the ",(0,r.jsx)(t.code,{children:"Recover"})," method.\nThe Coordinator recovers its key material and verifies the manifest history signature."]}),"\n",(0,r.jsx)(t.h2,{id:"workload-secrets",children:"Workload Secrets"}),"\n",(0,r.jsxs)(t.p,{children:["The Coordinator provides each workload a secret seed during attestation.\nThis secret can be used by the workload to derive additional secrets for example to encrypt persistent data.\nLike the workload certificates, it's written to the ",(0,r.jsx)(t.code,{children:"secrets/workload-secret-seed"})," path under the shared Kubernetes volume ",(0,r.jsx)(t.code,{children:"contrast-secrets"}),"."]}),"\n",(0,r.jsx)(t.admonition,{type:"warning",children:(0,r.jsx)(t.p,{children:"The workload owner can decrypt data encrypted with secrets derived from the workload secret.\nThe workload owner can derive the workload secret themselves, since it's derived from the secret seed known to the workload owner.\nIf the data owner and the workload owner is the same entity, then they can safely use the workload secrets."})}),"\n",(0,r.jsx)(t.h3,{id:"secure-persistence",children:"Secure persistence"}),"\n",(0,r.jsxs)(t.p,{children:["Remember that persistent volumes from the cloud provider are untrusted.\nUsing the built-in ",(0,r.jsx)(t.code,{children:"cryptsetup"})," subcommand of the initializer, applications can set up trusted storage on top of untrusted block devices based on the workload secret.\nFunctionally the initializer will act as a sidecar container which serves to set up a secure mount inside an ",(0,r.jsx)(t.code,{children:"emptyDir"})," mount shared with the main container."]}),"\n",(0,r.jsxs)(t.h4,{id:"usage-cryptsetup-subcommand",children:["Usage ",(0,r.jsx)(t.code,{children:"cryptsetup"})," subcommand"]}),"\n",(0,r.jsxs)(t.p,{children:["The ",(0,r.jsx)(t.code,{children:"cryptsetup"})," subcommand takes two arguments ",(0,r.jsx)(t.code,{children:"cryptsetup -d [device-path] -m [mount-point]"}),", to set up a LUKS-encrypted volume at ",(0,r.jsx)(t.code,{children:"device-path"})," and mount that volume at ",(0,r.jsx)(t.code,{children:"mount-point"}),"."]}),"\n",(0,r.jsx)(t.p,{children:"The following, slightly abbreviated resource outlines how this could be realized:"}),"\n",(0,r.jsx)(t.admonition,{type:"warning",children:(0,r.jsx)(t.p,{children:"This configuration snippet is intended to be educational and needs to be refined and adapted to your production environment.\nUsing it as-is may result in data corruption or data loss."})}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-yaml",children:'apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  name: volume-tester\nspec:\n  template:\n    spec:\n      containers:\n        - name: main\n          image: my.registry/my-image@sha256:0123... # <-- Original application requiring encrypted disk.\n          volumeMounts:\n            - mountPath: /state\n              mountPropagation: HostToContainer\n              name: share\n      initContainers:\n        - args:\n            - cryptsetup # <-- cryptsetup subcommand provided as args to the initializer binary.\n            - "--device-path"\n            - /dev/csi0\n            - "--mount-point"\n            - /state\n          image: "ghcr.io/edgelesssys/contrast/initializer:v1.4.1@sha256:b9a571e169cee0d04d382e2c63bb17c7e263a74bf946698fa677201f797a9fb1"\n          name: encrypted-volume-initializer\n          resources:\n            limits:\n              memory: 100Mi\n            requests:\n              memory: 100Mi\n          restartPolicy: Always\n          securityContext:\n            privileged: true # <-- This is necessary for mounting devices.\n          startupProbe:\n            exec:\n              command:\n                - /bin/test\n                - "-f"\n                - /done\n            failureThreshold: 20\n            periodSeconds: 5\n          volumeDevices:\n            - devicePath: /dev/csi0\n              name: state\n          volumeMounts:\n            - mountPath: /state\n              mountPropagation: Bidirectional\n              name: share\n      runtimeClassName: contrast-cc\n  volumeClaimTemplates:\n    - apiVersion: v1\n      kind: PersistentVolumeClaim\n      metadata:\n        name: state\n      spec:\n        accessModes:\n          - ReadWriteOnce\n        resources:\n          requests:\n            storage: 1Gi\n        volumeMode: Block # <-- The requested volume needs to be a raw block device.\n'})})]})}function l(e={}){const{wrapper:t}={...(0,i.R)(),...e.components};return t?(0,r.jsx)(t,{...e,children:(0,r.jsx)(h,{...e})}):h(e)}},28453:(e,t,n)=>{n.d(t,{R:()=>o,x:()=>a});var s=n(96540);const r={},i=s.createContext(r);function o(e){const t=s.useContext(i);return s.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function a(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:o(e.components),s.createElement(i.Provider,{value:t},e.children)}}}]);
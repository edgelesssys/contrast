"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[7682],{887:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>s,default:()=>h,frontMatter:()=>r,metadata:()=>o,toc:()=>d});var i=n(4848),a=n(8453);const r={},s="Contrast security overview",o={id:"basics/security-benefits",title:"Contrast security overview",description:"This document outlines the security measures of Contrast and its capability to counter various threats effectively.",source:"@site/docs/basics/security-benefits.md",sourceDirName:"basics",slug:"/basics/security-benefits",permalink:"/contrast/pr-preview/pr-400/next/basics/security-benefits",draft:!1,unlisted:!1,tags:[],version:"current",frontMatter:{},sidebar:"docs",previous:{title:"Confidential Containers",permalink:"/contrast/pr-preview/pr-400/next/basics/confidential-containers"},next:{title:"Product Features",permalink:"/contrast/pr-preview/pr-400/next/basics/features"}},c={},d=[{value:"Components of a Contrast deployment",id:"components-of-a-contrast-deployment",level:2},{value:"Examples of Contrasts use cases",id:"examples-of-contrasts-use-cases",level:2},{value:"Protecting the integrity and confidentiality of a workload",id:"protecting-the-integrity-and-confidentiality-of-a-workload",level:2},{value:"Attestation process",id:"attestation-process",level:3},{value:"Threat model and mitigations",id:"threat-model-and-mitigations",level:2},{value:"Possible attacks",id:"possible-attacks",level:3},{value:"Attack surfaces",id:"attack-surfaces",level:3},{value:"Threats and mitigations",id:"threats-and-mitigations",level:3},{value:"Attacks on the confidential container environment",id:"attacks-on-the-confidential-container-environment",level:4},{value:"Attacks on the Coordinator attestation service",id:"attacks-on-the-coordinator-attestation-service",level:4},{value:"Attacks on workloads",id:"attacks-on-workloads",level:4}];function l(e){const t={a:"a",em:"em",h1:"h1",h2:"h2",h3:"h3",h4:"h4",img:"img",li:"li",p:"p",strong:"strong",table:"table",tbody:"tbody",td:"td",th:"th",thead:"thead",tr:"tr",ul:"ul",...(0,a.R)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(t.h1,{id:"contrast-security-overview",children:"Contrast security overview"}),"\n",(0,i.jsx)(t.p,{children:"This document outlines the security measures of Contrast and its capability to counter various threats effectively.\nContrast is designed to shield entire Kubernetes deployments from the infrastructure, enabling entities to manage sensitive information (such as regulated or personally identifiable information (PII)) in the public cloud, while maintaining data confidentiality and ownership."}),"\n",(0,i.jsx)(t.p,{children:"Contrast is applicable in situations where establishing trust with the workload operator or the underlying infrastructure is challenging.\nThis is particularly beneficial for regulated sectors looking to transition sensitive activities to the cloud, without sacrificing security or compliance.\nIt allows for cloud adoption by maintaining a hardware-based separation from the cloud service provider."}),"\n",(0,i.jsx)(t.h2,{id:"components-of-a-contrast-deployment",children:"Components of a Contrast deployment"}),"\n",(0,i.jsxs)(t.p,{children:["Contrast implements the ",(0,i.jsx)(t.a,{href:"/contrast/pr-preview/pr-400/next/basics/confidential-containers",children:"Confidential Containers"})," concept.\nConfidential Containers significantly decrease the size of the trusted computing base (TCB) of a Kubernetes deployment, by isolating each pod within its own confidential micro-VM environment.\nThe TCB is the totality of elements in a computing environment that must be trusted not to be compromised.\nA smaller TCB results in a smaller attack surface. The following diagram shows how Confidential Containers remove the ",(0,i.jsx)(t.em,{children:"cloud & datacenter infrastructure"})," and the ",(0,i.jsx)(t.em,{children:"physical hosts"}),", including the hypervisor, the host OS, the Kubernetes control plane, and other components, from the TCB (red).\nIn the confidential context, represented by green, only the workload containers along with their confidential micro-VM environment are included within the Trusted Computing Base (TCB).\nTheir integrity is attested and can be ",(0,i.jsx)(t.a,{href:"/contrast/pr-preview/pr-400/next/architecture/attestation/hardware",children:"verified"}),"."]}),"\n",(0,i.jsxs)(t.p,{children:["Confidential Containers use ",(0,i.jsx)(t.a,{href:"/contrast/pr-preview/pr-400/next/basics/confidential-containers",children:"hardware-based mechanisms"}),", specifically leveraging CPU features, to ensure the isolation of the confidential context.\nThis implies that both the CPU and its microcode are integral components of the TCB.\nHowever, it should be noted that the hardware aspects aren't depicted in the accompanying graphic."]}),"\n",(0,i.jsx)(t.p,{children:(0,i.jsx)(t.img,{alt:"TCB comparison",src:n(7976).A+"",width:"3983",height:"1380"})}),"\n",(0,i.jsx)(t.p,{children:"A Contrast deployment has five core components:"}),"\n",(0,i.jsxs)(t.ul,{children:["\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.strong,{children:"The workload containers"}),": Container images that run in isolated Confidential Container environments."]}),"\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.strong,{children:"The runtime policies"}),": Policies that define the runtime environments for the workload containers."]}),"\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.strong,{children:"The manifest"}),": A manifest file defining the reference values of an entire confidential deployment. It contains the policy hashes for all pods of the deployment and the expected hardware reference values for the Confidential Container runtime."]}),"\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.strong,{children:"The Coordinator"}),": An attestation service that runs in a Confidential Container in the Kubernetes cluster. The Coordinator is configured with the manifest. User-facing, you can verify this service and the effective manifest using remote attestation, providing you with a concise attestation for the entire deployment. Cluster-facing, it verifies all pods and their policies based on remote attestation procedures and the manifest."]}),"\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.strong,{children:"The protected data"}),": The data that's processed by the workload containers."]}),"\n"]}),"\n",(0,i.jsx)(t.p,{children:"Contrast helps protect the workload and its runtime environment's integrity and confidentiality from inspection and tampering.\nFurthermore, it provides the ability to attest this isolation and the workloads identity for the entire distributed application at any point in time in a single operation."}),"\n",(0,i.jsx)(t.p,{children:"In a Contrast deployment, there are three parties:"}),"\n",(0,i.jsxs)(t.ul,{children:["\n",(0,i.jsxs)(t.li,{children:["\n",(0,i.jsxs)(t.p,{children:[(0,i.jsx)(t.strong,{children:"The container image provider"}),", who creates the container images that represent the application that has access to the protected data."]}),"\n"]}),"\n",(0,i.jsxs)(t.li,{children:["\n",(0,i.jsxs)(t.p,{children:[(0,i.jsx)(t.strong,{children:"The workload operator"}),", who runs the workload in a Kubernetes cluster. The operator typically has full administrative privileges to the deployment. The operator can manage cluster resources such as nodes, volumes, and networking rules, and the operator can interact with any Kubernetes or underlying cloud API."]}),"\n"]}),"\n",(0,i.jsxs)(t.li,{children:["\n",(0,i.jsxs)(t.p,{children:[(0,i.jsx)(t.strong,{children:"The data owner"}),", who owns the protected data. A data owner can verify the deployment using the Coordinator attestation service. The verification includes the identity, integrity, and confidentiality of the workloads, the runtime environment and the access permissions."]}),"\n"]}),"\n"]}),"\n",(0,i.jsx)(t.p,{children:"Contrast supports a trust model where the container image provider, workload operator, and data owner are separate, mutually distrusting parties."}),"\n",(0,i.jsx)(t.p,{children:"The following diagram shows the system components and parties."}),"\n",(0,i.jsx)(t.p,{children:(0,i.jsx)(t.img,{alt:"Components and parties",src:n(2766).A+"",width:"3588",height:"2017"})}),"\n",(0,i.jsx)(t.h2,{id:"examples-of-contrasts-use-cases",children:"Examples of Contrasts use cases"}),"\n",(0,i.jsx)(t.p,{children:"Contrast helps you to isolate your workloads and data from the infrastructure and the cloud service provider. The following table describes three example use cases."}),"\n",(0,i.jsxs)(t.table,{children:[(0,i.jsx)(t.thead,{children:(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.th,{children:"Use Case"}),(0,i.jsx)(t.th,{children:"Example Scenario"})]})}),(0,i.jsxs)(t.tbody,{children:[(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"Migrate sensitive workloads to the cloud"}),(0,i.jsx)(t.td,{children:"TechSolve Inc., a software development firm, aimed to enhance its defense-in-depth strategy for its cloud-based development environment, especially for projects involving proprietary algorithms and client data. Contrast adds robust, multi-layered security that protects proprietary algorithms and client data against a wide range of threats, including infrastructure-based attacks and insider risks."})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"Make your SaaS more trustworthy"}),(0,i.jsx)(t.td,{children:"SaaSProviderX, a company offering cloud-based project management tools, sought to enhance its platform's trustworthiness amidst growing concerns about data breaches and privacy. Contrasts adds a new layer of verification and isolation to bolster its platform security, increased user trust, and solidified its reputation as a secure and reliable SaaS provider."})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"Simplify regulatory compliance"}),(0,i.jsx)(t.td,{children:"HealthSecure Inc. has been managing a significant volume of sensitive patient data on-premises. With the increasing demand for advanced data analytics and the need for scalable infrastructure, the firm decides to migrate its data analytics operations to the cloud. However, the primary concern is maintaining the confidentiality and security of patient data during and after the migration, in compliance with healthcare regulations. Contrasts helps leveraging confidential containers that ensure compliance and maintain data confidentiality, thereby enhancing their analytics capabilities securely."})]})]})]}),"\n",(0,i.jsx)(t.p,{children:"In each scenario, Contrast ensures exclusive data access and processing capabilities are confined to the designated workloads. It achieves this by effectively isolating the workload from the infrastructure and other components of the stack. Data owners are granted the capability to audit and approve the deployment environment before submitting their data, ensuring a secure handover. Meanwhile, workload operators are equipped to manage and operate the application seamlessly, without requiring direct access to either the workload or its associated data."}),"\n",(0,i.jsx)(t.h2,{id:"protecting-the-integrity-and-confidentiality-of-a-workload",children:"Protecting the integrity and confidentiality of a workload"}),"\n",(0,i.jsx)(t.p,{children:"To help protect the workload from an untrusted workload operator and the infrastructure, Contrast implements the following security controls:"}),"\n",(0,i.jsxs)(t.ul,{children:["\n",(0,i.jsx)(t.li,{children:"An attestation process detects modifications to the workload image or its confidential container. This control helps protect the workload's integrity pre-attestation."}),"\n",(0,i.jsx)(t.li,{children:"A runtime policy and prevents the workload operator from accessing or compromising the instance at runtime. This control protects a workload's integrity and confidentiality post-attestation."}),"\n"]}),"\n",(0,i.jsx)(t.h3,{id:"attestation-process",children:"Attestation process"}),"\n",(0,i.jsxs)(t.p,{children:["The ",(0,i.jsx)(t.a,{href:"/contrast/pr-preview/pr-400/next/architecture/attestation/hardware",children:"attestation architecture"})," describes Contrast's chain of trust and the attestation process in detail."]}),"\n",(0,i.jsx)(t.h2,{id:"threat-model-and-mitigations",children:"Threat model and mitigations"}),"\n",(0,i.jsx)(t.p,{children:"This section describes the threat vectors that Contrast helps to mitigate."}),"\n",(0,i.jsx)(t.p,{children:"The following attacks are out of scope for this document:"}),"\n",(0,i.jsxs)(t.ul,{children:["\n",(0,i.jsx)(t.li,{children:"Attacks on the application code itself, such as insufficient access controls."}),"\n",(0,i.jsx)(t.li,{children:"Attacks on the Confidential Computing hardware directly, such as side-channel attacks."}),"\n",(0,i.jsx)(t.li,{children:"Attacks on the availability, such as denial-of-service (DOS) attacks."}),"\n"]}),"\n",(0,i.jsx)(t.h3,{id:"possible-attacks",children:"Possible attacks"}),"\n",(0,i.jsx)(t.p,{children:"Contrast is designed to defend against five possible attacks:"}),"\n",(0,i.jsxs)(t.ul,{children:["\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.strong,{children:"A malicious cloud insider"}),": malicious employees or third-party contractors of cloud service providers (CSPs) potentially have full access to various layers of the cloud infrastructure. That goes from the physical datacenter up to the hypervisor and Kubernetes layer. For example, they can access the physical memory of the machines, modify the hypervisor, modify disk contents, intercept network communications, and attempt to compromise the confidential container at runtime. A malicious insider can expand the attack surface or restrict the runtime environment. For example, a malicious operator can add a storage device to introduce new attack vectors. As another example, a malicious operator can constrain resources such as limiting a guest's memory size, changing its disk space, or changing firewall rules."]}),"\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.strong,{children:"A malicious cloud co-tenant"}),': malicious cloud user ("hackers") may break out of their tenancy and access other tenants\' data. Advanced attackers may even be able to establish a permanent foothold within the infrastructure and access data over a longer period. The threats are analogous to the ',(0,i.jsx)(t.em,{children:"cloud insider access"})," scenario, without the physical access."]}),"\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.strong,{children:"A malicious workload operator"}),": malicious workload operators, for example Kubernetes administrators, have full access to the workload deployment and the underlying Kubernetes platform. The threats are analogously to the ",(0,i.jsx)(t.em,{children:"cloud insider access"})," scenario, with access to everything that's above the hypervisor level."]}),"\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.strong,{children:"A malicious attestation client"}),": this attacker connects to the attestation service and sends malformed request."]}),"\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.strong,{children:"A malicious container image provider"}),": a malicious container image provider has full control over the application development itself. This attacker might release a malicious version of the workload containing harmful operations."]}),"\n"]}),"\n",(0,i.jsx)(t.h3,{id:"attack-surfaces",children:"Attack surfaces"}),"\n",(0,i.jsx)(t.p,{children:"The following table describes the attack surfaces that are available to attackers."}),"\n",(0,i.jsxs)(t.table,{children:[(0,i.jsx)(t.thead,{children:(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.th,{children:"Attacker"}),(0,i.jsx)(t.th,{children:"Target"}),(0,i.jsx)(t.th,{children:"Attack surface"}),(0,i.jsx)(t.th,{children:"Risks"})]})}),(0,i.jsxs)(t.tbody,{children:[(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"Cloud insider"}),(0,i.jsx)(t.td,{children:"Confidential Container, Workload"}),(0,i.jsx)(t.td,{children:"Physical memory"}),(0,i.jsx)(t.td,{children:"Attacker can dump the physical memory of the workloads."})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"Cloud insider, cloud hacker, workload operator"}),(0,i.jsx)(t.td,{children:"Confidential Container, Workload"}),(0,i.jsx)(t.td,{children:"Disk reads"}),(0,i.jsx)(t.td,{children:"Anything read from the disk is within the attacker's control."})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"Cloud insider, cloud hacker, workload operator"}),(0,i.jsx)(t.td,{children:"Confidential Container, Workload"}),(0,i.jsx)(t.td,{children:"Disk writes"}),(0,i.jsx)(t.td,{children:"Anything written to disk is visible to an attacker."})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"Cloud insider, cloud hacker, workload operator"}),(0,i.jsx)(t.td,{children:"Confidential Container, Workload"}),(0,i.jsx)(t.td,{children:"Kubernetes Control Plane"}),(0,i.jsx)(t.td,{children:"Instance attributes read from the Kubernetes control plane, including mount points and environment variables, are within the attacker's control."})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"Cloud insider, cloud hacker, workload operator"}),(0,i.jsx)(t.td,{children:"Confidential Container, Workload"}),(0,i.jsx)(t.td,{children:"Container Runtime"}),(0,i.jsx)(t.td,{children:'The attacker can use container runtime APIs (for example "kubectl exec") to perform operations on the workload container.'})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"Cloud insider, cloud hacker, workload operator"}),(0,i.jsx)(t.td,{children:"Confidential Container, Workload"}),(0,i.jsx)(t.td,{children:"Network"}),(0,i.jsx)(t.td,{children:"Intra-deployment (between containers) as well as external network connections to the image repository or attestation service can be intercepted."})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"Attestation client"}),(0,i.jsx)(t.td,{children:"Coordinator attestation service"}),(0,i.jsx)(t.td,{children:"Attestation requests"}),(0,i.jsx)(t.td,{children:"The attestation service has complex, crypto-heavy logic that's challenging to write defensively."})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"Container image provider"}),(0,i.jsx)(t.td,{children:"Workload"}),(0,i.jsx)(t.td,{children:"Workload"}),(0,i.jsx)(t.td,{children:"This attacker might release an upgrade to the workload containing harmful changes, such as a backdoor."})]})]})]}),"\n",(0,i.jsx)(t.h3,{id:"threats-and-mitigations",children:"Threats and mitigations"}),"\n",(0,i.jsx)(t.p,{children:"The container root file system with integrity protection is designed to mitigate risks from disk attacks.\nAdditionally, the container has no writeable disk partition mounted, hence, data is only stored in-memory and never disclosed to disk."}),"\n",(0,i.jsxs)(t.p,{children:["Risks from network attacks are mitigated by having ",(0,i.jsx)(t.a,{href:"/contrast/pr-preview/pr-400/next/components/service-mesh",children:"authenticated, end-to-end encrypted channels"}),".\nAn ",(0,i.jsx)(t.a,{href:"/contrast/pr-preview/pr-400/next/architecture/attestation/hardware",children:"attestation protocol"})," helps protect the boot sequence.\n",(0,i.jsx)(t.a,{href:"/contrast/pr-preview/pr-400/next/architecture/attestation/runtime-policies",children:"Runtime policies"})," verify the runtime environment configuration read from the Kubernetes control plane."]}),"\n",(0,i.jsx)(t.p,{children:"The following tables describe the threats and mitigations:"}),"\n",(0,i.jsxs)(t.ul,{children:["\n",(0,i.jsx)(t.li,{children:"Attacks on the confidential container environment"}),"\n",(0,i.jsx)(t.li,{children:"Attacks on the attestation service"}),"\n",(0,i.jsx)(t.li,{children:"Attacks on workloads"}),"\n"]}),"\n",(0,i.jsx)(t.h4,{id:"attacks-on-the-confidential-container-environment",children:"Attacks on the confidential container environment"}),"\n",(0,i.jsx)(t.p,{children:"This table describes potential threats and mitigation strategies related to the confidential container environment."}),"\n",(0,i.jsxs)(t.table,{children:[(0,i.jsx)(t.thead,{children:(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.th,{children:"Threat"}),(0,i.jsx)(t.th,{children:"Mitigation"}),(0,i.jsx)(t.th,{children:"Mitigation implementation"})]})}),(0,i.jsxs)(t.tbody,{children:[(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"An attacker intercepts the network connection of the launcher or image repository."}),(0,i.jsx)(t.td,{children:"An attacker can change the image URL and control the workload binary. However these actions are reflected in the attestation report. The image repository isn't controlled using an access list, therefore the image is assumed to be viewable by everyone. You must ensure that the workload container image doesn't contain any secrets."}),(0,i.jsx)(t.td,{children:"Within the Contrast container image"})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"An attacker modifies the workload image on disk after it was downloaded and measured."}),(0,i.jsx)(t.td,{children:"This threat is mitigated by a read-only partition that's integrity-protected. The workload image is protected by dm-verity."}),(0,i.jsx)(t.td,{children:"Within the Contrast pod VM image"})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"An attacker modifies a container's runtime environment configuration in the Kubernetes control plane."}),(0,i.jsx)(t.td,{children:"The attestation process and the runtime policies detects unsafe configurations that load non-authentic images or perform any other modification to the expected runtime environment."}),(0,i.jsx)(t.td,{children:"Within the runtime policies"})]})]})]}),"\n",(0,i.jsx)(t.h4,{id:"attacks-on-the-coordinator-attestation-service",children:"Attacks on the Coordinator attestation service"}),"\n",(0,i.jsx)(t.p,{children:"This table describes potential threats and mitigation strategies to the attestation service."}),"\n",(0,i.jsxs)(t.table,{children:[(0,i.jsx)(t.thead,{children:(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.th,{children:"Threat"}),(0,i.jsx)(t.th,{children:"Mitigation"}),(0,i.jsx)(t.th,{children:"Mitigation implementation"})]})}),(0,i.jsxs)(t.tbody,{children:[(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"An attacker intercepts the Coordinator deployment and modifies the image or hijacks the runtime environment."}),(0,i.jsx)(t.td,{children:"This threat is mitigated by having an attestation procedure and attested, encrypted TLS connections to the Coordinator. The attestation evidence for the Coordinator image is distributed with our releases, protected by supply chain security and fully reproducible."}),(0,i.jsx)(t.td,{children:"Within the CLI, and the Coordinator"})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"An attacker intercepts the network connection between the workload and the Coordinator and reads secret keys from the wire."}),(0,i.jsx)(t.td,{children:"This threat is mitigated by having an attested, encrypted TLS connection. This connection helps protect the token from passive eavesdropping. The attacker can't create valid workload certificates that would be accepted in Contrast's service mesh. An attacker can't impersonate a valid workload container because the container's identity is guaranteed by the attestation protocol."}),(0,i.jsx)(t.td,{children:"Within the network between your workload and the Coordinator."})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"An attacker exploits parsing discrepancies, which leads to undetected changes in the attestation process."}),(0,i.jsx)(t.td,{children:"This risk is mitigated by having a parsing engine written in memory-safe Go that's tested against the attestation specification of the hardware vendor. The runtime policies are available as an attestation artifact for further inspection and audits to verify their effectiveness."}),(0,i.jsx)(t.td,{children:"Within the Coordinator"})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"An attacker uses all service resources, which brings the Coordinator down in a denial of service (DoS) attack."}),(0,i.jsx)(t.td,{children:"This reliability risk is mitigated by having a distributed, Coordinator service that can be easily replicated and scaled out as needed."}),(0,i.jsx)(t.td,{children:"Within the Coordinator"})]})]})]}),"\n",(0,i.jsx)(t.h4,{id:"attacks-on-workloads",children:"Attacks on workloads"}),"\n",(0,i.jsx)(t.p,{children:"This table describes potential threats and mitigation strategies related to workloads."}),"\n",(0,i.jsxs)(t.table,{children:[(0,i.jsx)(t.thead,{children:(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.th,{children:"Threat"}),(0,i.jsx)(t.th,{children:"Mitigation"}),(0,i.jsx)(t.th,{children:"Mitigation implementation"})]})}),(0,i.jsxs)(t.tbody,{children:[(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"An attacker intercepts the network connection between two workload containers."}),(0,i.jsx)(t.td,{children:"This threat is mitigated by having transparently encrypted TLS connections between the containers in your deployment."}),(0,i.jsx)(t.td,{children:"Within the Contrast container image"})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"An attacker reads or modifies data written to disk via persistent volumes."}),(0,i.jsx)(t.td,{children:"Currently persistent volumes aren't supported in Contrast. In the future, this threat is mitigated by encrypted and integrity-protected volume mounts."}),(0,i.jsx)(t.td,{children:"Within the Contrast container image"})]}),(0,i.jsxs)(t.tr,{children:[(0,i.jsx)(t.td,{children:"An attacker publishes a new image version containing malicious code."}),(0,i.jsx)(t.td,{children:"The attestation process and the runtime policies require a data owner to accept a specific version of the workload and any update to the workload needs to be explicitly acknowledged."}),(0,i.jsx)(t.td,{children:"Within the attestation procedure"})]})]})]})]})}function h(e={}){const{wrapper:t}={...(0,a.R)(),...e.components};return t?(0,i.jsx)(t,{...e,children:(0,i.jsx)(l,{...e})}):l(e)}},2766:(e,t,n)=>{n.d(t,{A:()=>i});const i=n.p+"assets/images/personas-1c9f2b217e0b94e0c3057df96d96b3f0.svg"},7976:(e,t,n)=>{n.d(t,{A:()=>i});const i=n.p+"assets/images/tcb-89bc6e00f74de788d4a70ce25e55a927.svg"},8453:(e,t,n)=>{n.d(t,{R:()=>s,x:()=>o});var i=n(6540);const a={},r=i.createContext(a);function s(e){const t=i.useContext(r);return i.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function o(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(a):e.components||a:s(e.components),i.createElement(r.Provider,{value:t},e.children)}}}]);
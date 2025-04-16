"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[1555],{13950:(e,t,i)=>{i.r(t),i.d(t,{assets:()=>c,contentTitle:()=>h,default:()=>o,frontMatter:()=>d,metadata:()=>n,toc:()=>l});const n=JSON.parse('{"id":"architecture/snp","title":"AMD SEV-SNP attestation","description":"This page explains details of the AMD SEV-SNP attestation that are especially relevant for the implementation in Contrast.","source":"@site/docs/architecture/snp.md","sourceDirName":"architecture","slug":"/architecture/snp","permalink":"/contrast/pr-preview/pr-1396/next/architecture/snp","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/docs/architecture/snp.md","tags":[],"version":"current","frontMatter":{},"sidebar":"docs","previous":{"title":"Observability","permalink":"/contrast/pr-preview/pr-1396/next/architecture/observability"},"next":{"title":"Registry authentication","permalink":"/contrast/pr-preview/pr-1396/next/howto/registry-authentication"}}');var s=i(74848),r=i(28453);const d={},h="AMD SEV-SNP attestation",c={},l=[{value:"Startup",id:"startup",level:2},{value:"ID block structure",id:"id-block-structure",level:3},{value:"ID auth structure",id:"id-auth-structure",level:3},{value:"Guest policy structure",id:"guest-policy-structure",level:3},{value:"Attestation report",id:"attestation-report",level:2},{value:"Attestation report structure",id:"attestation-report-structure",level:3},{value:"Platform info structure",id:"platform-info-structure",level:3},{value:"Anonymous ID block signing",id:"anonymous-id-block-signing",level:2}];function a(e){const t={a:"a",code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",ol:"ol",p:"p",pre:"pre",table:"table",tbody:"tbody",td:"td",th:"th",thead:"thead",tr:"tr",...(0,r.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(t.header,{children:(0,s.jsx)(t.h1,{id:"amd-sev-snp-attestation",children:"AMD SEV-SNP attestation"})}),"\n",(0,s.jsxs)(t.p,{children:["This page explains details of the AMD SEV-SNP attestation that are especially relevant for the implementation in Contrast.\nYou should be familiar with ",(0,s.jsx)(t.a,{href:"/contrast/pr-preview/pr-1396/next/architecture/attestation",children:"attestation basics"}),".\nFor more details on SEV-SNP, see the ",(0,s.jsx)(t.a,{href:"https://www.amd.com/en/developer/sev.html",children:"whitepapers and specifications from AMD"}),"."]}),"\n",(0,s.jsx)(t.p,{children:"The key component for attesting AMD SEV-SNP is the Secure Processor (SP).\nIt measures the confidential VM (CVM) and metadata and returns an attestation report reflecting those\nmeasurements."}),"\n",(0,s.jsx)(t.h2,{id:"startup",children:"Startup"}),"\n",(0,s.jsxs)(t.p,{children:["The SP extends the launch digest each time the hypervisor donates a page to the CVM during startup via ",(0,s.jsx)(t.code,{children:"SNP_LAUNCH_UPDATE"}),". On an abstract level, the launch digest is extended as follows:"]}),"\n",(0,s.jsx)(t.pre,{children:(0,s.jsx)(t.code,{children:"LD := Hash(LD || Page || Page Metadata)\n"})}),"\n",(0,s.jsxs)(t.p,{children:["When the Hypervisor calls ",(0,s.jsx)(t.code,{children:"SNP_LAUNCH_FINISH"}),", it provides the SP with the ",(0,s.jsx)(t.code,{children:"HOST_DATA"}),",\nthe ",(0,s.jsx)(t.code,{children:"ID_BLOCK"}),", and ",(0,s.jsx)(t.code,{children:"ID_AUTH"})," block."]}),"\n",(0,s.jsxs)(t.p,{children:["The ",(0,s.jsx)(t.code,{children:"HOST_DATA"})," is opaque to the SP. Kata writes the hash of the kata policy in\nthis field to bind the policy to the CVM. ",(0,s.jsx)(t.code,{children:"HOST_DATA"})," is later reflected in the\nattestation report."]}),"\n",(0,s.jsx)(t.h3,{id:"id-block-structure",children:"ID block structure"}),"\n",(0,s.jsx)(t.p,{children:"The ID block contains fields that identify the VM.\nThe following table shows the fields that are relevant for Contrast.\nThe complete structure can be found in the SEV Secure Nested Paging Firmware ABI Specification in table 75."}),"\n",(0,s.jsxs)(t.table,{children:[(0,s.jsx)(t.thead,{children:(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.th,{children:"Field"}),(0,s.jsx)(t.th,{children:"Description"}),(0,s.jsx)(t.th,{children:"Contrast usage"})]})}),(0,s.jsxs)(t.tbody,{children:[(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"LD"}),(0,s.jsx)(t.td,{children:"The expected launch digest of the guest."}),(0,s.jsx)(t.td,{children:"Expected launch digest over kernel, initrd, and cmdline of the CVM."})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"POLICY"}),(0,s.jsx)(t.td,{children:"The policy of the guest."}),(0,s.jsx)(t.td,{})]})]})]}),"\n",(0,s.jsxs)(t.p,{children:["During startup, the SP compares the measurement it calculated to the ",(0,s.jsx)(t.code,{children:"LD"})," in the ID block.\nIf they don't match, the SP aborts the boot process.\nSimilarly, if the policy doesn't match the configuration of the CVM, the SP aborts also."]}),"\n",(0,s.jsx)(t.h3,{id:"id-auth-structure",children:"ID auth structure"}),"\n",(0,s.jsx)(t.p,{children:"The ID auth structure exists to be able to verify the ID block structure.\nA CVM image can be started with for example various different policies.\nMoreover, the ID block itself can't be verified later, since\nit's not part of the attestation report.\nThe intended use is that a trusted party creates an ECDSA-384 ID key pair\nand signs the ID block structure. Both the signature and the public part of the\nID key are then passed via the hypervisor to the SP. The SP verifies the signature and\nkeeps a SHA-384 digest of the public key. The SP adds this digest in every attestation\nreport requested by the CVM."}),"\n",(0,s.jsx)(t.p,{children:"The complete structure can be found in the SEV Secure Nested Paging Firmware ABI Specification in table 76."}),"\n",(0,s.jsxs)(t.table,{children:[(0,s.jsx)(t.thead,{children:(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.th,{children:"Field"}),(0,s.jsx)(t.th,{children:"Description"}),(0,s.jsx)(t.th,{children:"Contrast usage"})]})}),(0,s.jsxs)(t.tbody,{children:[(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"ID_BLOCK_SIG"}),(0,s.jsx)(t.td,{children:"The signature of all bytes of the ID block."}),(0,s.jsx)(t.td,{children:"Constant value of (r,s) = (2,1)"})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"ID_KEY"}),(0,s.jsx)(t.td,{children:"The public component of the ID key."}),(0,s.jsxs)(t.td,{children:["Deterministically derived from the ID block and ID_BLOCK_SIG (see ",(0,s.jsx)(t.a,{href:"#anonymous-id-block-signing",children:"Anonymous ID block signing"}),")"]})]})]})]}),"\n",(0,s.jsx)(t.h3,{id:"guest-policy-structure",children:"Guest policy structure"}),"\n",(0,s.jsxs)(t.p,{children:["The guest policy structure is embedded in the ID block in the ",(0,s.jsx)(t.code,{children:"POLICY"})," field.\nThe complete structure can be found in the SEV Secure Nested Paging Firmware ABI Specification in table 10."]}),"\n",(0,s.jsxs)(t.table,{children:[(0,s.jsx)(t.thead,{children:(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.th,{children:"Field"}),(0,s.jsx)(t.th,{children:"Description"}),(0,s.jsx)(t.th,{children:"Value on cloud-hypervisor"}),(0,s.jsx)(t.th,{children:"Value on QEMU"})]})}),(0,s.jsxs)(t.tbody,{children:[(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"CIPHERTEXT_HIDING_DRAM"}),(0,s.jsxs)(t.td,{children:["0: Ciphertext hiding for the DRAM may be enabled or disabled. ",(0,s.jsx)("br",{}),"1: Ciphertext hiding for the DRAM must be enabled."]}),(0,s.jsx)(t.td,{children:"0"}),(0,s.jsx)(t.td,{children:"0"})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"RAPL_DIS"}),(0,s.jsxs)(t.td,{children:["0: Allow Running Average Power Limit (RAPL).  ",(0,s.jsx)("br",{}),"1: RAPL must be disabled"]}),(0,s.jsx)(t.td,{children:"0"}),(0,s.jsx)(t.td,{children:"0"})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"MEM_AES_256_XTS"}),(0,s.jsxs)(t.td,{children:["0: Allow either AES 128 XEX or AES 256 XTS for memory encryption. ",(0,s.jsx)("br",{})," 1: Require AES 256 XTS for memory encryption."]}),(0,s.jsx)(t.td,{children:"0"}),(0,s.jsx)(t.td,{children:"0"})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"CXL_ALLOW"}),(0,s.jsxs)(t.td,{children:["0: CXL can't be populated with devices or memory. ",(0,s.jsx)("br",{})," 1: CXL can be populated with devices or memory."]}),(0,s.jsx)(t.td,{children:"0"}),(0,s.jsx)(t.td,{children:"0"})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"SINGLE_SOCKET"}),(0,s.jsxs)(t.td,{children:["0: Guest can be activated on multiple sockets. ",(0,s.jsx)("br",{})," 1: Guest can be activated only on one socket."]}),(0,s.jsx)(t.td,{children:"0"}),(0,s.jsx)(t.td,{children:"0"})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"DEBUG"}),(0,s.jsxs)(t.td,{children:["0: Debugging is disallowed. ",(0,s.jsx)("br",{}),"  1: Debugging is allowed"]}),(0,s.jsx)(t.td,{children:"0"}),(0,s.jsx)(t.td,{children:"0"})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"MIGRATE_MA"}),(0,s.jsxs)(t.td,{children:["0: Association with a migration agent is disallowed. ",(0,s.jsx)("br",{})," 1: Association with a migration agent is allowed."]}),(0,s.jsx)(t.td,{children:"0"}),(0,s.jsx)(t.td,{children:"0"})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"SMT"}),(0,s.jsxs)(t.td,{children:["0: SMT is disallowed. ",(0,s.jsx)("br",{})," 1: SMT is allowed."]}),(0,s.jsx)(t.td,{children:"1"}),(0,s.jsx)(t.td,{children:"1"})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"ABI_MAJOR"}),(0,s.jsx)(t.td,{children:"The minimum ABI major version required for this guest to run."}),(0,s.jsx)(t.td,{children:"0"}),(0,s.jsx)(t.td,{children:"0"})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"ABI_MINOR"}),(0,s.jsx)(t.td,{children:"The minimum ABI minor version required for this guest to run."}),(0,s.jsx)(t.td,{children:"0"}),(0,s.jsx)(t.td,{children:"31"})]})]})]}),"\n",(0,s.jsx)(t.h2,{id:"attestation-report",children:"Attestation report"}),"\n",(0,s.jsx)(t.p,{children:"The attestation report is signed by the Versioned Chip Endorsement Key (VCEK).\nThe SP derives this key from the chip unique secret and the following REPORTED_TCB\ninformation:"}),"\n",(0,s.jsxs)(t.ol,{children:["\n",(0,s.jsx)(t.li,{children:"SP bootloader SVN"}),"\n",(0,s.jsx)(t.li,{children:"SP OS SVN"}),"\n",(0,s.jsx)(t.li,{children:"SNP firmware SVN"}),"\n",(0,s.jsx)(t.li,{children:"Microcode patch level"}),"\n"]}),"\n",(0,s.jsxs)(t.p,{children:["With those parameters, one can retrieve a certificate signing the VCEK from the\nAMD Key Distribution Service (KDS) by querying ",(0,s.jsx)(t.code,{children:"https://kdsintf.amd.com/vcek/v1/{product_name}/{hwid}?{params}"})]}),"\n",(0,s.jsx)(t.p,{children:"This VCEK certificate is signed by the AMD SEV CA certificate, which is signed by the AMD Root CA."}),"\n",(0,s.jsx)(t.pre,{children:(0,s.jsx)(t.code,{children:"AMD Root CA --\x3e AMD SEV CA --\x3e VCEK -- signs --\x3e Report\n"})}),"\n",(0,s.jsx)(t.p,{children:"The Contrast CLI embeds the AMD Root CA and AMD SEV CA certificate."}),"\n",(0,s.jsx)(t.h3,{id:"attestation-report-structure",children:"Attestation report structure"}),"\n",(0,s.jsx)(t.p,{children:"The following table shows the most important fields of the attestation report and how they're used by Contrast.\nThe complete structure can be found in the SEV Secure Nested Paging Firmware ABI Specification in table 23."}),"\n",(0,s.jsxs)(t.table,{children:[(0,s.jsx)(t.thead,{children:(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.th,{children:"Field"}),(0,s.jsx)(t.th,{children:"Description"}),(0,s.jsx)(t.th,{children:"Contrast usage"})]})}),(0,s.jsxs)(t.tbody,{children:[(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"VERSION"}),(0,s.jsxs)(t.td,{children:["Version number of this attestation report. Set to ",(0,s.jsx)(t.code,{children:"3h"})," for this specification."]}),(0,s.jsx)(t.td,{})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"PLATFORM_INFO"}),(0,s.jsx)(t.td,{children:"Information about the platform. See Table below"}),(0,s.jsx)(t.td,{})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"REPORT_DATA"}),(0,s.jsx)(t.td,{children:"If REQUEST_SOURCE is guest provided, then contains Guest-provided data, else host request and zero (0) filled by firmware."}),(0,s.jsx)(t.td,{children:"Digest of nonce provided by the relying party and TLS public key of the CVM."})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"MEASUREMENT"}),(0,s.jsx)(t.td,{children:"The measurement calculated at launch."}),(0,s.jsx)(t.td,{children:"Digest over kernel, initrd, and cmdline."})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"HOST_DATA"}),(0,s.jsx)(t.td,{children:"Data provided by the hypervisor at launch."}),(0,s.jsx)(t.td,{children:"Digest of the kata policy."})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"ID_KEY_DIGEST"}),(0,s.jsx)(t.td,{children:"SHA-384 digest of the ID public key that signed the ID block provided in SNP_LAUNCH_FINISH."}),(0,s.jsxs)(t.td,{children:["Deterministic function of the SNP policy and launch digest in the ID_BLOCK. (see ",(0,s.jsx)(t.a,{href:"#anonymous-id-block-signing",children:"Anonymous ID block signing"}),")"]})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"LAUNCH_TCB"}),(0,s.jsx)(t.td,{children:"The CurrentTcb at the time the guest was launched or imported."}),(0,s.jsx)(t.td,{children:"Lowest TCB the guest ever executed with."})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"SIGNATURE"}),(0,s.jsxs)(t.td,{children:["Signature of bytes ",(0,s.jsx)(t.code,{children:"0h"})," to ",(0,s.jsx)(t.code,{children:"29Fh"})," inclusive of this report."]}),(0,s.jsx)(t.td,{children:"Used to verify the integrity and authenticity of the report."})]})]})]}),"\n",(0,s.jsx)(t.h3,{id:"platform-info-structure",children:"Platform info structure"}),"\n",(0,s.jsxs)(t.p,{children:["The platform info structure is embedded in the attestation report in the ",(0,s.jsx)(t.code,{children:"PLATFORM_INFO"})," field.\nThe complete structure can be found in the SEV Secure Nested Paging Firmware ABI Specification in table 24."]}),"\n",(0,s.jsxs)(t.table,{children:[(0,s.jsx)(t.thead,{children:(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.th,{children:"Field"}),(0,s.jsx)(t.th,{children:"Description"})]})}),(0,s.jsxs)(t.tbody,{children:[(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"ALIAS_CHECK_COMPLETE"}),(0,s.jsx)(t.td,{children:"Indicates that alias detection has completed since the last system reset and there are no aliasing addresses. Resets to 0. Contains mitigation for CVE-2024-21944."})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"CIPHERTEXT_HIDING_DRAM_EN"}),(0,s.jsx)(t.td,{children:"Indicates ciphertext hiding is enabled for the DRAM."})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"RAPL_DIS"}),(0,s.jsx)(t.td,{children:"Indicates that the RAPL feature is disabled."})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"ECC_EN"}),(0,s.jsx)(t.td,{children:"Indicates that the platform is using error correcting codes for memory. Present when EccMemReporting feature bit is set."})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"TSME_EN"}),(0,s.jsx)(t.td,{children:"Indicates that TSME is enabled in the system."})]}),(0,s.jsxs)(t.tr,{children:[(0,s.jsx)(t.td,{children:"SMT_EN"}),(0,s.jsx)(t.td,{children:"Indicates that SMT is enabled in the system."})]})]})]}),"\n",(0,s.jsx)(t.h2,{id:"anonymous-id-block-signing",children:"Anonymous ID block signing"}),"\n",(0,s.jsxs)(t.p,{children:["As described in ",(0,s.jsx)(t.a,{href:"#startup",children:"startup"}),", the SP checks the signature of the ID block\nwith the public key provided in the ID auth block. The common usage of such signatures\nis to know that a trusted party holding the private key has signed the ID block.\nSince the ID block is part of for example the ",(0,s.jsx)(t.a,{href:"https://github.com/microsoft/igvm",children:"IGVM"})," headers of\nthe VM image, they're bound to the ",(0,s.jsx)(t.code,{children:"runtimeClass"})," Contrast sets up\nduring ",(0,s.jsx)(t.a,{href:"/contrast/pr-preview/pr-1396/next/getting-started/install",children:"installation"}),".\nTherefore, the ID auth block and the signature and public key has to be provided by\nContrast, but the authors of Contrast shouldn't be part of the TCB."]}),"\n",(0,s.jsxs)(t.p,{children:["To have both the ability to sign ID blocks and not be part of the TCB, we must ensure\nthat there exists no private key for the ",(0,s.jsx)(t.code,{children:"ID_KEY"})," in the ID auth structure.\nFor this, we implement ECDSA public key recovery. The algorithm is defined in ",(0,s.jsx)(t.a,{href:"https://www.secg.org/sec1-v2.pdf",children:"SEC 1: Elliptic Curve Cryptography"}),".\nThe algorithm calculates an ECDSA public key given a message and its signature.\nWe keep the signature constant as ",(0,s.jsx)(t.code,{children:"(r,s) = (2,1)"})," for all versions and\nuse the given ID block containing the guest policy and launch digest as an input.\nThe recovery algorithm returns two valid public keys. We choose the smaller one, meaning\nthe one with the smaller x value and, if equal, the one with the smaller y value."]}),"\n",(0,s.jsx)(t.p,{children:"Since we don't generate any private key material during recovery and calculating the private\nkey from only the message, signature, and public key is cryptographically hard, no one\ncan forge (ID block, signature) combinations under the same public key."})]})}function o(e={}){const{wrapper:t}={...(0,r.R)(),...e.components};return t?(0,s.jsx)(t,{...e,children:(0,s.jsx)(a,{...e})}):a(e)}},28453:(e,t,i)=>{i.d(t,{R:()=>d,x:()=>h});var n=i(96540);const s={},r=n.createContext(s);function d(e){const t=n.useContext(r);return n.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function h(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:d(e.components),n.createElement(r.Provider,{value:t},e.children)}}}]);
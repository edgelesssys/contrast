"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[4670],{2585:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>a,contentTitle:()=>i,default:()=>m,frontMatter:()=>s,metadata:()=>l,toc:()=>u});var n=r(4848),c=r(8453),o=r(4074);const s={},i="Architecture",l={id:"architecture/index",title:"Architecture",description:"",source:"@site/versioned_docs/version-0.5/architecture/index.md",sourceDirName:"architecture",slug:"/architecture/",permalink:"/contrast/pr-preview/pr-577/0.5/architecture/",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.5/architecture/index.md",tags:[],version:"0.5",frontMatter:{},sidebar:"docs",previous:{title:"Workload deployment",permalink:"/contrast/pr-preview/pr-577/0.5/deployment"},next:{title:"Components",permalink:"/contrast/pr-preview/pr-577/0.5/category/components"}},a={},u=[];function d(e){const t={h1:"h1",...(0,c.R)(),...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:"architecture",children:"Architecture"}),"\n","\n",(0,n.jsx)(o.A,{})]})}function m(e={}){const{wrapper:t}={...(0,c.R)(),...e.components};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(d,{...e})}):d(e)}},4074:(e,t,r)=>{r.d(t,{A:()=>C});var n=r(6540),c=r(4164),o=r(5215),s=r(4783),i=r(7639);const l=["zero","one","two","few","many","other"];function a(e){return l.filter((t=>e.includes(t)))}const u={locale:"en",pluralForms:a(["one","other"]),select:e=>1===e?"one":"other"};function d(){const{i18n:{currentLocale:e}}=(0,i.A)();return(0,n.useMemo)((()=>{try{return function(e){const t=new Intl.PluralRules(e);return{locale:e,pluralForms:a(t.resolvedOptions().pluralCategories),select:e=>t.select(e)}}(e)}catch(t){return console.error(`Failed to use Intl.PluralRules for locale "${e}".\nDocusaurus will fallback to the default (English) implementation.\nError: ${t.message}\n`),u}}),[e])}function m(){const e=d();return{selectMessage:(t,r)=>function(e,t,r){const n=e.split("|");if(1===n.length)return n[0];n.length>r.pluralForms.length&&console.error(`For locale=${r.locale}, a maximum of ${r.pluralForms.length} plural forms are expected (${r.pluralForms.join(",")}), but the message contains ${n.length}: ${e}`);const c=r.select(t),o=r.pluralForms.indexOf(c);return n[Math.min(o,n.length-1)]}(r,t,e)}}var p=r(877),h=r(3230),f=r(5225);const x={cardContainer:"cardContainer_fWXF",cardTitle:"cardTitle_rnsV",cardDescription:"cardDescription_PWke"};var g=r(4848);function j(e){let{href:t,children:r}=e;return(0,g.jsx)(s.A,{href:t,className:(0,c.A)("card padding--lg",x.cardContainer),children:r})}function v(e){let{href:t,icon:r,title:n,description:o}=e;return(0,g.jsxs)(j,{href:t,children:[(0,g.jsxs)(f.A,{as:"h2",className:(0,c.A)("text--truncate",x.cardTitle),title:n,children:[r," ",n]}),o&&(0,g.jsx)("p",{className:(0,c.A)("text--truncate",x.cardDescription),title:o,children:o})]})}function w(e){let{item:t}=e;const r=(0,o.Nr)(t),n=function(){const{selectMessage:e}=m();return t=>e(t,(0,h.T)({message:"{count} items",id:"theme.docs.DocCard.categoryDescription.plurals",description:"The default description for a category card in the generated index about how many items this category includes"},{count:t}))}();return r?(0,g.jsx)(v,{href:r,icon:"\ud83d\uddc3\ufe0f",title:t.label,description:t.description??n(t.items.length)}):null}function y(e){let{item:t}=e;const r=(0,p.A)(t.href)?"\ud83d\udcc4\ufe0f":"\ud83d\udd17",n=(0,o.cC)(t.docId??void 0);return(0,g.jsx)(v,{href:t.href,icon:r,title:t.label,description:t.description??n?.description})}function A(e){let{item:t}=e;switch(t.type){case"link":return(0,g.jsx)(y,{item:t});case"category":return(0,g.jsx)(w,{item:t});default:throw new Error(`unknown item type ${JSON.stringify(t)}`)}}function k(e){let{className:t}=e;const r=(0,o.$S)();return(0,g.jsx)(C,{items:r.items,className:t})}function C(e){const{items:t,className:r}=e;if(!t)return(0,g.jsx)(k,{...e});const n=(0,o.d1)(t);return(0,g.jsx)("section",{className:(0,c.A)("row",r),children:n.map(((e,t)=>(0,g.jsx)("article",{className:"col col--6 margin-bottom--lg",children:(0,g.jsx)(A,{item:e})},t)))})}},8453:(e,t,r)=>{r.d(t,{R:()=>s,x:()=>i});var n=r(6540);const c={},o=n.createContext(c);function s(e){const t=n.useContext(o);return n.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function i(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(c):e.components||c:s(e.components),n.createElement(o.Provider,{value:t},e.children)}}}]);
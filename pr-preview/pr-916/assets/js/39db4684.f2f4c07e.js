"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[3459],{3941:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>l,contentTitle:()=>i,default:()=>m,frontMatter:()=>c,metadata:()=>a,toc:()=>u});var r=n(4848),o=n(8453),s=n(4074);const c={},i="About",a={id:"about/index",title:"About",description:"",source:"@site/versioned_docs/version-0.7/about/index.md",sourceDirName:"about",slug:"/about/",permalink:"/contrast/pr-preview/pr-916/0.7/about/",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.7/about/index.md",tags:[],version:"0.7",frontMatter:{},sidebar:"docs",previous:{title:"Planned features and limitations",permalink:"/contrast/pr-preview/pr-916/0.7/features-limitations"},next:{title:"Telemetry",permalink:"/contrast/pr-preview/pr-916/0.7/about/telemetry"}},l={},u=[];function d(e){const t={h1:"h1",header:"header",...(0,o.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(t.header,{children:(0,r.jsx)(t.h1,{id:"about",children:"About"})}),"\n","\n",(0,r.jsx)(s.A,{})]})}function m(e={}){const{wrapper:t}={...(0,o.R)(),...e.components};return t?(0,r.jsx)(t,{...e,children:(0,r.jsx)(d,{...e})}):d(e)}},4074:(e,t,n)=>{n.d(t,{A:()=>k});var r=n(6540),o=n(4164),s=n(5357),c=n(4783),i=n(7639);const a=["zero","one","two","few","many","other"];function l(e){return a.filter((t=>e.includes(t)))}const u={locale:"en",pluralForms:l(["one","other"]),select:e=>1===e?"one":"other"};function d(){const{i18n:{currentLocale:e}}=(0,i.A)();return(0,r.useMemo)((()=>{try{return function(e){const t=new Intl.PluralRules(e);return{locale:e,pluralForms:l(t.resolvedOptions().pluralCategories),select:e=>t.select(e)}}(e)}catch(t){return console.error(`Failed to use Intl.PluralRules for locale "${e}".\nDocusaurus will fallback to the default (English) implementation.\nError: ${t.message}\n`),u}}),[e])}function m(){const e=d();return{selectMessage:(t,n)=>function(e,t,n){const r=e.split("|");if(1===r.length)return r[0];r.length>n.pluralForms.length&&console.error(`For locale=${n.locale}, a maximum of ${n.pluralForms.length} plural forms are expected (${n.pluralForms.join(",")}), but the message contains ${r.length}: ${e}`);const o=n.select(t),s=n.pluralForms.indexOf(o);return r[Math.min(s,r.length-1)]}(n,t,e)}}var p=n(877),h=n(3230),f=n(5225);const x={cardContainer:"cardContainer_fWXF",cardTitle:"cardTitle_rnsV",cardDescription:"cardDescription_PWke"};var g=n(4848);function b(e){let{href:t,children:n}=e;return(0,g.jsx)(c.A,{href:t,className:(0,o.A)("card padding--lg",x.cardContainer),children:n})}function j(e){let{href:t,icon:n,title:r,description:s}=e;return(0,g.jsxs)(b,{href:t,children:[(0,g.jsxs)(f.A,{as:"h2",className:(0,o.A)("text--truncate",x.cardTitle),title:r,children:[n," ",r]}),s&&(0,g.jsx)("p",{className:(0,o.A)("text--truncate",x.cardDescription),title:s,children:s})]})}function v(e){let{item:t}=e;const n=(0,s.Nr)(t),r=function(){const{selectMessage:e}=m();return t=>e(t,(0,h.T)({message:"1 item|{count} items",id:"theme.docs.DocCard.categoryDescription.plurals",description:"The default description for a category card in the generated index about how many items this category includes"},{count:t}))}();return n?(0,g.jsx)(j,{href:n,icon:"\ud83d\uddc3\ufe0f",title:t.label,description:t.description??r(t.items.length)}):null}function w(e){let{item:t}=e;const n=(0,p.A)(t.href)?"\ud83d\udcc4\ufe0f":"\ud83d\udd17",r=(0,s.cC)(t.docId??void 0);return(0,g.jsx)(j,{href:t.href,icon:n,title:t.label,description:t.description??r?.description})}function y(e){let{item:t}=e;switch(t.type){case"link":return(0,g.jsx)(w,{item:t});case"category":return(0,g.jsx)(v,{item:t});default:throw new Error(`unknown item type ${JSON.stringify(t)}`)}}function A(e){let{className:t}=e;const n=(0,s.$S)();return(0,g.jsx)(k,{items:n.items,className:t})}function k(e){const{items:t,className:n}=e;if(!t)return(0,g.jsx)(A,{...e});const r=(0,s.d1)(t);return(0,g.jsx)("section",{className:(0,o.A)("row",n),children:r.map(((e,t)=>(0,g.jsx)("article",{className:"col col--6 margin-bottom--lg",children:(0,g.jsx)(y,{item:e})},t)))})}},8453:(e,t,n)=>{n.d(t,{R:()=>c,x:()=>i});var r=n(6540);const o={},s=r.createContext(o);function c(e){const t=r.useContext(s);return r.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function i(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(o):e.components||o:c(e.components),r.createElement(s.Provider,{value:t},e.children)}}}]);
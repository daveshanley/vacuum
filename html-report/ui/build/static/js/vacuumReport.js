/*! For license information please see vacuumReport.js.LICENSE.txt */
(()=>{"use strict";var t={408:(t,e,o)=>{o.d(e,{Z:()=>l});var r=o(81),i=o.n(r),n=o(645),a=o.n(n)()(i());a.push([t.id,':root{--global-font-size:15px;--global-line-height:1.4em;--global-space:10px;--font-stack:Menlo,Monaco,Lucida Console,Liberation Mono,DejaVu Sans Mono,Bitstream Vera Sans Mono,Courier New,monospace,serif;--mono-font-stack:Menlo,Monaco,Lucida Console,Liberation Mono,DejaVu Sans Mono,Bitstream Vera Sans Mono,Courier New,monospace,serif;--background-color:#fff;--page-width:60em;--font-color:#151515;--invert-font-color:#fff;--primary-color:#1a95e0;--secondary-color:#727578;--error-color:#d20962;--progress-bar-background:#727578;--progress-bar-fill:#151515;--code-bg-color:#e8eff2;--input-style:solid;--display-h1-decoration:none}*{box-sizing:border-box;text-rendering:geometricPrecision}::-moz-selection{background:var(--primary-color);color:var(--invert-font-color)}::selection{background:var(--primary-color);color:var(--invert-font-color)}body{font-size:var(--global-font-size);color:var(--font-color);line-height:var(--global-line-height);margin:0;font-family:var(--font-stack);word-wrap:break-word;background-color:var(--background-color)}.logo,h1,h2,h3,h4,h5,h6{line-height:var(--global-line-height)}a{cursor:pointer;color:var(--primary-color);text-decoration:none}a:hover{background-color:var(--primary-color);color:var(--invert-font-color)}em{font-size:var(--global-font-size);font-style:italic;font-family:var(--font-stack);color:var(--font-color)}blockquote,code,em,strong{line-height:var(--global-line-height)}.logo,blockquote,code,footer,h1,h2,h3,h4,h5,h6,header,li,ol,p,section,ul{float:none;margin:0;padding:0}.logo,blockquote,h1,ol,p,ul{margin-top:calc(var(--global-space) * 2);margin-bottom:calc(var(--global-space) * 2)}.logo,h1{position:relative;display:inline-block;display:table-cell;padding:calc(var(--global-space) * 2) 0 calc(var(--global-space) * 2);margin:0;overflow:hidden;font-weight:600}h1::after{content:"====================================================================================================";position:absolute;bottom:5px;left:0;display:var(--display-h1-decoration)}.logo+*,h1+*{margin-top:0}h2,h3,h4,h5,h6{position:relative;margin-bottom:var(--global-line-height);font-weight:600}blockquote{position:relative;padding-left:calc(var(--global-space) * 2);padding-left:2ch;overflow:hidden}blockquote::after{content:">\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>";white-space:pre;position:absolute;top:0;left:0;line-height:var(--global-line-height);color:#9ca2ab}code{font-weight:inherit;background-color:var(--code-bg-color);font-family:var(--mono-font-stack)}code::after,code::before{content:"`";display:inline}pre code::after,pre code::before{content:""}pre{display:block;word-break:break-all;word-wrap:break-word;color:var(--secondary-color);background-color:var(--background-color);border:1px solid var(--secondary-color);padding:var(--global-space);white-space:pre-wrap;white-space:-moz-pre-wrap;white-space:-pre-wrap;white-space:-o-pre-wrap}pre code{overflow-x:scroll;padding:0;margin:0;display:inline-block;min-width:100%;font-family:var(--mono-font-stack)}.terminal .logo,.terminal blockquote,.terminal code,.terminal h1,.terminal h2,.terminal h3,.terminal h4,.terminal h5,.terminal h6,.terminal strong{font-size:var(--global-font-size);font-style:normal;font-family:var(--font-stack);color:var(--font-color)}.terminal-prompt{position:relative;white-space:nowrap}.terminal-prompt::before{content:"> "}.terminal-prompt::after{content:"";-webkit-animation:cursor .8s infinite;animation:cursor .8s infinite;background:var(--primary-color);border-radius:0;display:inline-block;height:1em;margin-left:.2em;width:3px;bottom:-2px;position:relative}@-webkit-keyframes cursor{0%{opacity:0}50%{opacity:1}to{opacity:0}}@keyframes cursor{0%{opacity:0}50%{opacity:1}to{opacity:0}}li,li>ul>li{position:relative;display:block;padding-left:calc(var(--global-space) * 2)}nav>ul>li{padding-left:0}li::after{position:absolute;top:0;left:0}ul>li::after{content:"-"}nav ul>li::after{content:""}ol li::before{content:counters(item, ".") ". ";counter-increment:item}ol ol li::before{content:counters(item, ".") " ";counter-increment:item}.terminal-menu li::after,.terminal-menu li::before{display:none}ol{counter-reset:item}ol li:nth-child(n+10)::after{left:-7px}ol ol{margin-top:0;margin-bottom:0}.terminal-menu{width:100%}.terminal-nav{display:flex;flex-direction:column;align-items:flex-start}ul ul{margin-top:0;margin-bottom:0}.terminal-menu ul{list-style-type:none;padding:0!important;display:flex;flex-direction:column;width:100%;flex-grow:1;font-size:var(--global-font-size);margin-top:0}.terminal-menu li{display:flex;margin:0 0 .5em 0;padding:0}ol.terminal-toc li{border-bottom:1px dotted var(--secondary-color);padding:0;margin-bottom:15px}.terminal-menu li:last-child{margin-bottom:0}ol.terminal-toc li a{margin:4px 4px 4px 0;background:var(--background-color);position:relative;top:6px;text-align:left;padding-right:4px}.terminal-menu li a:not(.btn){text-decoration:none;display:block;width:100%;border:none;color:var(--secondary-color)}.terminal-menu li a.active{color:var(--font-color)}.terminal-menu li a:hover{background:0 0;color:inherit}ol.terminal-toc li::before{content:counters(item, ".") ". ";counter-increment:item;position:absolute;right:0;background:var(--background-color);padding:4px 0 4px 4px;bottom:-8px}ol.terminal-toc li a:hover{background:var(--primary-color);color:var(--invert-font-color)}hr{position:relative;overflow:hidden;margin:calc(var(--global-space) * 4) 0;border:0;border-bottom:1px dashed var(--secondary-color)}p{margin:0 0 var(--global-line-height);color:var(--global-font-color)}.container{max-width:var(--page-width)}.container,.container-fluid{margin:0 auto;padding:0 calc(var(--global-space) * 2)}img{width:100%}.progress-bar{height:8px;background-color:var(--progress-bar-background);margin:12px 0}.progress-bar.progress-bar-show-percent{margin-top:38px}.progress-bar-filled{background-color:var(--progress-bar-fill);height:100%;transition:width .3s ease;position:relative;width:0}.progress-bar-filled::before{content:"";border:6px solid transparent;border-top-color:var(--progress-bar-fill);position:absolute;top:-12px;right:-6px}.progress-bar-filled::after{color:var(--progress-bar-fill);content:attr(data-filled);display:block;font-size:12px;white-space:nowrap;position:absolute;border:6px solid transparent;top:-38px;right:0;transform:translateX(50%)}.progress-bar-no-arrow>.progress-bar-filled::after,.progress-bar-no-arrow>.progress-bar-filled::before{content:"";display:none;visibility:hidden;opacity:0}table{width:100%;border-collapse:collapse;margin:var(--global-line-height) 0;color:var(--font-color);font-size:var(--global-font-size)}table td,table th{vertical-align:top;border:1px solid var(--font-color);line-height:var(--global-line-height);padding:calc(var(--global-space)/ 2);font-size:1em}table thead th{font-size:1em}table tfoot tr th{font-weight:500}table caption{font-size:1em;margin:0 0 1em 0}table tbody td:first-child{font-weight:700;color:var(--secondary-color)}.form{width:100%}fieldset{border:1px solid var(--font-color);padding:1em}label{font-size:1em;color:var(--font-color)}input[type=email],input[type=number],input[type=password],input[type=search],input[type=text]{border:1px var(--input-style) var(--font-color);width:100%;padding:.7em .5em;font-size:1em;font-family:var(--font-stack);-webkit-appearance:none;border-radius:0}input[type=email]:active,input[type=email]:focus,input[type=number]:active,input[type=number]:focus,input[type=password]:active,input[type=password]:focus,input[type=search]:active,input[type=search]:focus,input[type=text]:active,input[type=text]:focus{outline:0;-webkit-appearance:none;border:1px solid var(--font-color)}input[type=email]:not(:placeholder-shown):invalid,input[type=number]:not(:placeholder-shown):invalid,input[type=password]:not(:placeholder-shown):invalid,input[type=search]:not(:placeholder-shown):invalid,input[type=text]:not(:placeholder-shown):invalid{border-color:var(--error-color)}input,textarea{color:var(--font-color);background-color:var(--background-color)}input::-webkit-input-placeholder,textarea::-webkit-input-placeholder{color:var(--secondary-color)!important;opacity:1}input::-moz-placeholder,textarea::-moz-placeholder{color:var(--secondary-color)!important;opacity:1}input:-ms-input-placeholder,textarea:-ms-input-placeholder{color:var(--secondary-color)!important;opacity:1}input::-ms-input-placeholder,textarea::-ms-input-placeholder{color:var(--secondary-color)!important;opacity:1}input::placeholder,textarea::placeholder{color:var(--secondary-color)!important;opacity:1}textarea{height:auto;width:100%;resize:none;border:1px var(--input-style) var(--font-color);padding:.5em;font-size:1em;font-family:var(--font-stack);-webkit-appearance:none;border-radius:0}textarea:focus{outline:0;-webkit-appearance:none;border:1px solid var(--font-color)}textarea:not(:placeholder-shown):invalid{border-color:var(--error-color)}input:-webkit-autofill,input:-webkit-autofill:focus textarea:-webkit-autofill,input:-webkit-autofill:hover,select:-webkit-autofill,select:-webkit-autofill:focus,select:-webkit-autofill:hover,textarea:-webkit-autofill:hover textarea:-webkit-autofill:focus{border:1px solid var(--font-color);-webkit-text-fill-color:var(--font-color);box-shadow:0 0 0 1000px var(--invert-font-color) inset;-webkit-box-shadow:0 0 0 1000px var(--invert-font-color) inset;transition:background-color 5000s ease-in-out 0s}.form-group{margin-bottom:var(--global-line-height);overflow:auto}.btn{border-style:solid;border-width:1px;display:inline-flex;align-items:center;justify-content:center;cursor:pointer;outline:0;padding:.65em 2em;font-size:1em;font-family:inherit;-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;position:relative;z-index:1}.btn:active{box-shadow:none}.btn.btn-ghost{border-color:var(--font-color);color:var(--font-color);background-color:transparent}.btn.btn-ghost:focus,.btn.btn-ghost:hover{border-color:var(--tertiary-color);color:var(--tertiary-color);z-index:2}.btn.btn-ghost:hover{background-color:transparent}.btn-block{width:100%;display:flex}.btn-default{background-color:var(--font-color);border-color:var(--invert-font-color);color:var(--invert-font-color)}.btn-default:focus:not(.btn-ghost),.btn-default:hover{background-color:var(--secondary-color);color:var(--invert-font-color)}.btn-default.btn-ghost:focus,.btn-default.btn-ghost:hover{border-color:var(--secondary-color);color:var(--secondary-color);z-index:2}.btn-error{color:var(--invert-font-color);background-color:var(--error-color);border:1px solid var(--error-color)}.btn-error:focus:not(.btn-ghost),.btn-error:hover{background-color:var(--error-color);border-color:var(--error-color)}.btn-error.btn-ghost{border-color:var(--error-color);color:var(--error-color)}.btn-error.btn-ghost:focus,.btn-error.btn-ghost:hover{border-color:var(--error-color);color:var(--error-color);z-index:2}.btn-primary{color:var(--invert-font-color);background-color:var(--primary-color);border:1px solid var(--primary-color)}.btn-primary:focus:not(.btn-ghost),.btn-primary:hover{background-color:var(--primary-color);border-color:var(--primary-color)}.btn-primary.btn-ghost{border-color:var(--primary-color);color:var(--primary-color)}.btn-primary.btn-ghost:focus,.btn-primary.btn-ghost:hover{border-color:var(--primary-color);color:var(--primary-color);z-index:2}.btn-small{padding:.5em 1.3em!important;font-size:.9em!important}.btn-group{overflow:auto}.btn-group .btn{float:left}.btn-group .btn-ghost:not(:first-child){margin-left:-1px}.terminal-card{border:1px solid var(--secondary-color)}.terminal-card>header{color:var(--invert-font-color);text-align:center;background-color:var(--secondary-color);padding:.5em 0}.terminal-card>div:first-of-type{padding:var(--global-space)}.terminal-timeline{position:relative;padding-left:70px}.terminal-timeline::before{content:\' \';background:var(--secondary-color);display:inline-block;position:absolute;left:35px;width:2px;height:100%;z-index:400}.terminal-timeline .terminal-card{margin-bottom:25px}.terminal-timeline .terminal-card::before{content:\' \';background:var(--invert-font-color);border:2px solid var(--secondary-color);display:inline-block;position:absolute;margin-top:25px;left:26px;width:15px;height:15px;z-index:400}.terminal-alert{color:var(--font-color);padding:1em;border:1px solid var(--font-color);margin-bottom:var(--global-space)}.terminal-alert-error{color:var(--error-color);border-color:var(--error-color)}.terminal-alert-primary{color:var(--primary-color);border-color:var(--primary-color)}@media screen and (max-width:960px){label{display:block;width:100%}pre::-webkit-scrollbar{height:3px}}@media screen and (max-width:480px){form{width:100%}}@media only screen and (min-width:30em){.terminal-nav{flex-direction:row;align-items:center}.terminal-menu ul{flex-direction:row;justify-items:flex-end;align-items:center;justify-content:flex-end;margin-top:calc(var(--global-space) * 2)}.terminal-menu li{margin:0;margin-right:2em}.terminal-menu li:last-child{margin-right:0}}.terminal-media:not(:last-child){margin-bottom:1.25rem}.terminal-media-left{padding-right:var(--global-space)}.terminal-media-left,.terminal-media-right{display:table-cell;vertical-align:top}.terminal-media-right{padding-left:var(--global-space)}.terminal-media-body{display:table-cell;vertical-align:top}.terminal-media-heading{font-size:1em;font-weight:700}.terminal-media-content{margin-top:.3rem}.terminal-placeholder{background-color:var(--secondary-color);text-align:center;color:var(--font-color);font-size:1rem;border:1px solid var(--secondary-color)}figure>img{padding:0}.terminal-avatarholder{width:calc(var(--global-space) * 5);height:calc(var(--global-space) * 5)}.terminal-avatarholder img{padding:0}figure{margin:0}figure>figcaption{color:var(--secondary-color);text-align:center}.hljs{display:block;overflow-x:auto;padding:.5em;background:var(--block-background-color);color:var(--font-color)}.hljs-comment,.hljs-quote{color:var(--secondary-color)}.hljs-variable{color:var(--font-color)}.hljs-built_in,.hljs-keyword,.hljs-name,.hljs-selector-tag,.hljs-tag{color:var(--primary-color)}.hljs-addition,.hljs-attribute,.hljs-literal,.hljs-section,.hljs-string,.hljs-template-tag,.hljs-template-variable,.hljs-title,.hljs-type{color:var(--secondary-color)}.hljs-string{color:var(--secondary-color)}.hljs-deletion,.hljs-meta,.hljs-selector-attr,.hljs-selector-pseudo{color:var(--primary-color)}.hljs-doctag{color:var(--secondary-color)}.hljs-attr{color:var(--primary-color)}.hljs-bullet,.hljs-link,.hljs-symbol{color:var(--primary-color)}.hljs-emphasis{font-style:italic}.hljs-strong{font-weight:700}',""]);const l=a},645:t=>{t.exports=function(t){var e=[];return e.toString=function(){return this.map((function(e){var o="",r=void 0!==e[5];return e[4]&&(o+="@supports (".concat(e[4],") {")),e[2]&&(o+="@media ".concat(e[2]," {")),r&&(o+="@layer".concat(e[5].length>0?" ".concat(e[5]):""," {")),o+=t(e),r&&(o+="}"),e[2]&&(o+="}"),e[4]&&(o+="}"),o})).join("")},e.i=function(t,o,r,i,n){"string"==typeof t&&(t=[[null,t,void 0]]);var a={};if(r)for(var l=0;l<this.length;l++){var s=this[l][0];null!=s&&(a[s]=!0)}for(var c=0;c<t.length;c++){var d=[].concat(t[c]);r&&a[d[0]]||(void 0!==n&&(void 0===d[5]||(d[1]="@layer".concat(d[5].length>0?" ".concat(d[5]):""," {").concat(d[1],"}")),d[5]=n),o&&(d[2]?(d[1]="@media ".concat(d[2]," {").concat(d[1],"}"),d[2]=o):d[2]=o),i&&(d[4]?(d[1]="@supports (".concat(d[4],") {").concat(d[1],"}"),d[4]=i):d[4]="".concat(i)),e.push(d))}},e}},81:t=>{t.exports=function(t){return t[1]}},379:t=>{var e=[];function o(t){for(var o=-1,r=0;r<e.length;r++)if(e[r].identifier===t){o=r;break}return o}function r(t,r){for(var n={},a=[],l=0;l<t.length;l++){var s=t[l],c=r.base?s[0]+r.base:s[0],d=n[c]||0,h="".concat(c," ").concat(d);n[c]=d+1;var p=o(h),u={css:s[1],media:s[2],sourceMap:s[3],supports:s[4],layer:s[5]};if(-1!==p)e[p].references++,e[p].updater(u);else{var v=i(u,r);r.byIndex=l,e.splice(l,0,{identifier:h,updater:v,references:1})}a.push(h)}return a}function i(t,e){var o=e.domAPI(e);return o.update(t),function(e){if(e){if(e.css===t.css&&e.media===t.media&&e.sourceMap===t.sourceMap&&e.supports===t.supports&&e.layer===t.layer)return;o.update(t=e)}else o.remove()}}t.exports=function(t,i){var n=r(t=t||[],i=i||{});return function(t){t=t||[];for(var a=0;a<n.length;a++){var l=o(n[a]);e[l].references--}for(var s=r(t,i),c=0;c<n.length;c++){var d=o(n[c]);0===e[d].references&&(e[d].updater(),e.splice(d,1))}n=s}}},569:t=>{var e={};t.exports=function(t,o){var r=function(t){if(void 0===e[t]){var o=document.querySelector(t);if(window.HTMLIFrameElement&&o instanceof window.HTMLIFrameElement)try{o=o.contentDocument.head}catch(t){o=null}e[t]=o}return e[t]}(t);if(!r)throw new Error("Couldn't find a style target. This probably means that the value for the 'insert' parameter is invalid.");r.appendChild(o)}},216:t=>{t.exports=function(t){var e=document.createElement("style");return t.setAttributes(e,t.attributes),t.insert(e,t.options),e}},565:(t,e,o)=>{t.exports=function(t){var e=o.nc;e&&t.setAttribute("nonce",e)}},795:t=>{t.exports=function(t){var e=t.insertStyleElement(t);return{update:function(o){!function(t,e,o){var r="";o.supports&&(r+="@supports (".concat(o.supports,") {")),o.media&&(r+="@media ".concat(o.media," {"));var i=void 0!==o.layer;i&&(r+="@layer".concat(o.layer.length>0?" ".concat(o.layer):""," {")),r+=o.css,i&&(r+="}"),o.media&&(r+="}"),o.supports&&(r+="}");var n=o.sourceMap;n&&"undefined"!=typeof btoa&&(r+="\n/*# sourceMappingURL=data:application/json;base64,".concat(btoa(unescape(encodeURIComponent(JSON.stringify(n))))," */")),e.styleTagTransform(r,t,e.options)}(e,t,o)},remove:function(){!function(t){if(null===t.parentNode)return!1;t.parentNode.removeChild(t)}(e)}}}},589:t=>{t.exports=function(t,e){if(e.styleSheet)e.styleSheet.cssText=t;else{for(;e.firstChild;)e.removeChild(e.firstChild);e.appendChild(document.createTextNode(t))}}}},e={};function o(r){var i=e[r];if(void 0!==i)return i.exports;var n=e[r]={id:r,exports:{}};return t[r](n,n.exports,o),n.exports}o.n=t=>{var e=t&&t.__esModule?()=>t.default:()=>t;return o.d(e,{a:e}),e},o.d=(t,e)=>{for(var r in e)o.o(e,r)&&!o.o(t,r)&&Object.defineProperty(t,r,{enumerable:!0,get:e[r]})},o.o=(t,e)=>Object.prototype.hasOwnProperty.call(t,e),o.nc=void 0,(()=>{var t=o(379),e=o.n(t),r=o(795),i=o.n(r),n=o(569),a=o.n(n),l=o(565),s=o.n(l),c=o(216),d=o.n(c),h=o(589),p=o.n(h),u=o(408),v={};v.styleTagTransform=p(),v.setAttributes=s(),v.insert=a().bind(null,"head"),v.domAPI=i(),v.insertStyleElement=d(),e()(u.Z,v),u.Z&&u.Z.locals&&u.Z.locals;const m=window,f=m.ShadowRoot&&(void 0===m.ShadyCSS||m.ShadyCSS.nativeShadow)&&"adoptedStyleSheets"in Document.prototype&&"replace"in CSSStyleSheet.prototype,g=Symbol(),b=new WeakMap;class y{constructor(t,e,o){if(this._$cssResult$=!0,o!==g)throw Error("CSSResult is not constructable. Use `unsafeCSS` or `css` instead.");this.cssText=t,this.t=e}get styleSheet(){let t=this.o;const e=this.t;if(f&&void 0===t){const o=void 0!==e&&1===e.length;o&&(t=b.get(e)),void 0===t&&((this.o=t=new CSSStyleSheet).replaceSync(this.cssText),o&&b.set(e,t))}return t}toString(){return this.cssText}}const $=f?t=>t:t=>t instanceof CSSStyleSheet?(t=>{let e="";for(const o of t.cssRules)e+=o.cssText;return(t=>new y("string"==typeof t?t:t+"",void 0,g))(e)})(t):t;var A;const w=window,_=w.trustedTypes,x=_?_.emptyScript:"",S=w.reactiveElementPolyfillSupport,k={toAttribute(t,e){switch(e){case Boolean:t=t?x:null;break;case Object:case Array:t=null==t?t:JSON.stringify(t)}return t},fromAttribute(t,e){let o=t;switch(e){case Boolean:o=null!==t;break;case Number:o=null===t?null:Number(t);break;case Object:case Array:try{o=JSON.parse(t)}catch(t){o=null}}return o}},E=(t,e)=>e!==t&&(e==e||t==t),C={attribute:!0,type:String,converter:k,reflect:!1,hasChanged:E};class P extends HTMLElement{constructor(){super(),this._$Ei=new Map,this.isUpdatePending=!1,this.hasUpdated=!1,this._$El=null,this.u()}static addInitializer(t){var e;this.finalize(),(null!==(e=this.h)&&void 0!==e?e:this.h=[]).push(t)}static get observedAttributes(){this.finalize();const t=[];return this.elementProperties.forEach(((e,o)=>{const r=this._$Ep(o,e);void 0!==r&&(this._$Ev.set(r,o),t.push(r))})),t}static createProperty(t,e=C){if(e.state&&(e.attribute=!1),this.finalize(),this.elementProperties.set(t,e),!e.noAccessor&&!this.prototype.hasOwnProperty(t)){const o="symbol"==typeof t?Symbol():"__"+t,r=this.getPropertyDescriptor(t,o,e);void 0!==r&&Object.defineProperty(this.prototype,t,r)}}static getPropertyDescriptor(t,e,o){return{get(){return this[e]},set(r){const i=this[t];this[e]=r,this.requestUpdate(t,i,o)},configurable:!0,enumerable:!0}}static getPropertyOptions(t){return this.elementProperties.get(t)||C}static finalize(){if(this.hasOwnProperty("finalized"))return!1;this.finalized=!0;const t=Object.getPrototypeOf(this);if(t.finalize(),void 0!==t.h&&(this.h=[...t.h]),this.elementProperties=new Map(t.elementProperties),this._$Ev=new Map,this.hasOwnProperty("properties")){const t=this.properties,e=[...Object.getOwnPropertyNames(t),...Object.getOwnPropertySymbols(t)];for(const o of e)this.createProperty(o,t[o])}return this.elementStyles=this.finalizeStyles(this.styles),!0}static finalizeStyles(t){const e=[];if(Array.isArray(t)){const o=new Set(t.flat(1/0).reverse());for(const t of o)e.unshift($(t))}else void 0!==t&&e.push($(t));return e}static _$Ep(t,e){const o=e.attribute;return!1===o?void 0:"string"==typeof o?o:"string"==typeof t?t.toLowerCase():void 0}u(){var t;this._$E_=new Promise((t=>this.enableUpdating=t)),this._$AL=new Map,this._$Eg(),this.requestUpdate(),null===(t=this.constructor.h)||void 0===t||t.forEach((t=>t(this)))}addController(t){var e,o;(null!==(e=this._$ES)&&void 0!==e?e:this._$ES=[]).push(t),void 0!==this.renderRoot&&this.isConnected&&(null===(o=t.hostConnected)||void 0===o||o.call(t))}removeController(t){var e;null===(e=this._$ES)||void 0===e||e.splice(this._$ES.indexOf(t)>>>0,1)}_$Eg(){this.constructor.elementProperties.forEach(((t,e)=>{this.hasOwnProperty(e)&&(this._$Ei.set(e,this[e]),delete this[e])}))}createRenderRoot(){var t;const e=null!==(t=this.shadowRoot)&&void 0!==t?t:this.attachShadow(this.constructor.shadowRootOptions);return((t,e)=>{f?t.adoptedStyleSheets=e.map((t=>t instanceof CSSStyleSheet?t:t.styleSheet)):e.forEach((e=>{const o=document.createElement("style"),r=m.litNonce;void 0!==r&&o.setAttribute("nonce",r),o.textContent=e.cssText,t.appendChild(o)}))})(e,this.constructor.elementStyles),e}connectedCallback(){var t;void 0===this.renderRoot&&(this.renderRoot=this.createRenderRoot()),this.enableUpdating(!0),null===(t=this._$ES)||void 0===t||t.forEach((t=>{var e;return null===(e=t.hostConnected)||void 0===e?void 0:e.call(t)}))}enableUpdating(t){}disconnectedCallback(){var t;null===(t=this._$ES)||void 0===t||t.forEach((t=>{var e;return null===(e=t.hostDisconnected)||void 0===e?void 0:e.call(t)}))}attributeChangedCallback(t,e,o){this._$AK(t,o)}_$EO(t,e,o=C){var r;const i=this.constructor._$Ep(t,o);if(void 0!==i&&!0===o.reflect){const n=(void 0!==(null===(r=o.converter)||void 0===r?void 0:r.toAttribute)?o.converter:k).toAttribute(e,o.type);this._$El=t,null==n?this.removeAttribute(i):this.setAttribute(i,n),this._$El=null}}_$AK(t,e){var o;const r=this.constructor,i=r._$Ev.get(t);if(void 0!==i&&this._$El!==i){const t=r.getPropertyOptions(i),n="function"==typeof t.converter?{fromAttribute:t.converter}:void 0!==(null===(o=t.converter)||void 0===o?void 0:o.fromAttribute)?t.converter:k;this._$El=i,this[i]=n.fromAttribute(e,t.type),this._$El=null}}requestUpdate(t,e,o){let r=!0;void 0!==t&&(((o=o||this.constructor.getPropertyOptions(t)).hasChanged||E)(this[t],e)?(this._$AL.has(t)||this._$AL.set(t,e),!0===o.reflect&&this._$El!==t&&(void 0===this._$EC&&(this._$EC=new Map),this._$EC.set(t,o))):r=!1),!this.isUpdatePending&&r&&(this._$E_=this._$Ej())}async _$Ej(){this.isUpdatePending=!0;try{await this._$E_}catch(t){Promise.reject(t)}const t=this.scheduleUpdate();return null!=t&&await t,!this.isUpdatePending}scheduleUpdate(){return this.performUpdate()}performUpdate(){var t;if(!this.isUpdatePending)return;this.hasUpdated,this._$Ei&&(this._$Ei.forEach(((t,e)=>this[e]=t)),this._$Ei=void 0);let e=!1;const o=this._$AL;try{e=this.shouldUpdate(o),e?(this.willUpdate(o),null===(t=this._$ES)||void 0===t||t.forEach((t=>{var e;return null===(e=t.hostUpdate)||void 0===e?void 0:e.call(t)})),this.update(o)):this._$Ek()}catch(t){throw e=!1,this._$Ek(),t}e&&this._$AE(o)}willUpdate(t){}_$AE(t){var e;null===(e=this._$ES)||void 0===e||e.forEach((t=>{var e;return null===(e=t.hostUpdated)||void 0===e?void 0:e.call(t)})),this.hasUpdated||(this.hasUpdated=!0,this.firstUpdated(t)),this.updated(t)}_$Ek(){this._$AL=new Map,this.isUpdatePending=!1}get updateComplete(){return this.getUpdateComplete()}getUpdateComplete(){return this._$E_}shouldUpdate(t){return!0}update(t){void 0!==this._$EC&&(this._$EC.forEach(((t,e)=>this._$EO(e,this[e],t))),this._$EC=void 0),this._$Ek()}updated(t){}firstUpdated(t){}}var N;P.finalized=!0,P.elementProperties=new Map,P.elementStyles=[],P.shadowRootOptions={mode:"open"},null==S||S({ReactiveElement:P}),(null!==(A=w.reactiveElementVersions)&&void 0!==A?A:w.reactiveElementVersions=[]).push("1.4.2");const z=window,O=z.trustedTypes,R=O?O.createPolicy("lit-html",{createHTML:t=>t}):void 0,U=`lit$${(Math.random()+"").slice(9)}$`,T="?"+U,j=`<${T}>`,H=document,L=(t="")=>H.createComment(t),M=t=>null===t||"object"!=typeof t&&"function"!=typeof t,I=Array.isArray,D=/<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g,q=/-->/g,B=/>/g,V=RegExp(">|[ \t\n\f\r](?:([^\\s\"'>=/]+)([ \t\n\f\r]*=[ \t\n\f\r]*(?:[^ \t\n\f\r\"'`<>=]|(\"|')|))|$)","g"),F=/'/g,G=/"/g,W=/^(?:script|style|textarea|title)$/i,K=t=>(e,...o)=>({_$litType$:t,strings:e,values:o}),Z=(K(1),K(2),Symbol.for("lit-noChange")),J=Symbol.for("lit-nothing"),X=new WeakMap,Q=H.createTreeWalker(H,129,null,!1),Y=(t,e)=>{const o=t.length-1,r=[];let i,n=2===e?"<svg>":"",a=D;for(let e=0;e<o;e++){const o=t[e];let l,s,c=-1,d=0;for(;d<o.length&&(a.lastIndex=d,s=a.exec(o),null!==s);)d=a.lastIndex,a===D?"!--"===s[1]?a=q:void 0!==s[1]?a=B:void 0!==s[2]?(W.test(s[2])&&(i=RegExp("</"+s[2],"g")),a=V):void 0!==s[3]&&(a=V):a===V?">"===s[0]?(a=null!=i?i:D,c=-1):void 0===s[1]?c=-2:(c=a.lastIndex-s[2].length,l=s[1],a=void 0===s[3]?V:'"'===s[3]?G:F):a===G||a===F?a=V:a===q||a===B?a=D:(a=V,i=void 0);const h=a===V&&t[e+1].startsWith("/>")?" ":"";n+=a===D?o+j:c>=0?(r.push(l),o.slice(0,c)+"$lit$"+o.slice(c)+U+h):o+U+(-2===c?(r.push(void 0),e):h)}const l=n+(t[o]||"<?>")+(2===e?"</svg>":"");if(!Array.isArray(t)||!t.hasOwnProperty("raw"))throw Error("invalid template strings array");return[void 0!==R?R.createHTML(l):l,r]};class tt{constructor({strings:t,_$litType$:e},o){let r;this.parts=[];let i=0,n=0;const a=t.length-1,l=this.parts,[s,c]=Y(t,e);if(this.el=tt.createElement(s,o),Q.currentNode=this.el.content,2===e){const t=this.el.content,e=t.firstChild;e.remove(),t.append(...e.childNodes)}for(;null!==(r=Q.nextNode())&&l.length<a;){if(1===r.nodeType){if(r.hasAttributes()){const t=[];for(const e of r.getAttributeNames())if(e.endsWith("$lit$")||e.startsWith(U)){const o=c[n++];if(t.push(e),void 0!==o){const t=r.getAttribute(o.toLowerCase()+"$lit$").split(U),e=/([.?@])?(.*)/.exec(o);l.push({type:1,index:i,name:e[2],strings:t,ctor:"."===e[1]?nt:"?"===e[1]?lt:"@"===e[1]?st:it})}else l.push({type:6,index:i})}for(const e of t)r.removeAttribute(e)}if(W.test(r.tagName)){const t=r.textContent.split(U),e=t.length-1;if(e>0){r.textContent=O?O.emptyScript:"";for(let o=0;o<e;o++)r.append(t[o],L()),Q.nextNode(),l.push({type:2,index:++i});r.append(t[e],L())}}}else if(8===r.nodeType)if(r.data===T)l.push({type:2,index:i});else{let t=-1;for(;-1!==(t=r.data.indexOf(U,t+1));)l.push({type:7,index:i}),t+=U.length-1}i++}}static createElement(t,e){const o=H.createElement("template");return o.innerHTML=t,o}}function et(t,e,o=t,r){var i,n,a,l;if(e===Z)return e;let s=void 0!==r?null===(i=o._$Co)||void 0===i?void 0:i[r]:o._$Cl;const c=M(e)?void 0:e._$litDirective$;return(null==s?void 0:s.constructor)!==c&&(null===(n=null==s?void 0:s._$AO)||void 0===n||n.call(s,!1),void 0===c?s=void 0:(s=new c(t),s._$AT(t,o,r)),void 0!==r?(null!==(a=(l=o)._$Co)&&void 0!==a?a:l._$Co=[])[r]=s:o._$Cl=s),void 0!==s&&(e=et(t,s._$AS(t,e.values),s,r)),e}class ot{constructor(t,e){this.u=[],this._$AN=void 0,this._$AD=t,this._$AM=e}get parentNode(){return this._$AM.parentNode}get _$AU(){return this._$AM._$AU}v(t){var e;const{el:{content:o},parts:r}=this._$AD,i=(null!==(e=null==t?void 0:t.creationScope)&&void 0!==e?e:H).importNode(o,!0);Q.currentNode=i;let n=Q.nextNode(),a=0,l=0,s=r[0];for(;void 0!==s;){if(a===s.index){let e;2===s.type?e=new rt(n,n.nextSibling,this,t):1===s.type?e=new s.ctor(n,s.name,s.strings,this,t):6===s.type&&(e=new ct(n,this,t)),this.u.push(e),s=r[++l]}a!==(null==s?void 0:s.index)&&(n=Q.nextNode(),a++)}return i}p(t){let e=0;for(const o of this.u)void 0!==o&&(void 0!==o.strings?(o._$AI(t,o,e),e+=o.strings.length-2):o._$AI(t[e])),e++}}class rt{constructor(t,e,o,r){var i;this.type=2,this._$AH=J,this._$AN=void 0,this._$AA=t,this._$AB=e,this._$AM=o,this.options=r,this._$Cm=null===(i=null==r?void 0:r.isConnected)||void 0===i||i}get _$AU(){var t,e;return null!==(e=null===(t=this._$AM)||void 0===t?void 0:t._$AU)&&void 0!==e?e:this._$Cm}get parentNode(){let t=this._$AA.parentNode;const e=this._$AM;return void 0!==e&&11===t.nodeType&&(t=e.parentNode),t}get startNode(){return this._$AA}get endNode(){return this._$AB}_$AI(t,e=this){t=et(this,t,e),M(t)?t===J||null==t||""===t?(this._$AH!==J&&this._$AR(),this._$AH=J):t!==this._$AH&&t!==Z&&this.g(t):void 0!==t._$litType$?this.$(t):void 0!==t.nodeType?this.T(t):(t=>I(t)||"function"==typeof(null==t?void 0:t[Symbol.iterator]))(t)?this.k(t):this.g(t)}O(t,e=this._$AB){return this._$AA.parentNode.insertBefore(t,e)}T(t){this._$AH!==t&&(this._$AR(),this._$AH=this.O(t))}g(t){this._$AH!==J&&M(this._$AH)?this._$AA.nextSibling.data=t:this.T(H.createTextNode(t)),this._$AH=t}$(t){var e;const{values:o,_$litType$:r}=t,i="number"==typeof r?this._$AC(t):(void 0===r.el&&(r.el=tt.createElement(r.h,this.options)),r);if((null===(e=this._$AH)||void 0===e?void 0:e._$AD)===i)this._$AH.p(o);else{const t=new ot(i,this),e=t.v(this.options);t.p(o),this.T(e),this._$AH=t}}_$AC(t){let e=X.get(t.strings);return void 0===e&&X.set(t.strings,e=new tt(t)),e}k(t){I(this._$AH)||(this._$AH=[],this._$AR());const e=this._$AH;let o,r=0;for(const i of t)r===e.length?e.push(o=new rt(this.O(L()),this.O(L()),this,this.options)):o=e[r],o._$AI(i),r++;r<e.length&&(this._$AR(o&&o._$AB.nextSibling,r),e.length=r)}_$AR(t=this._$AA.nextSibling,e){var o;for(null===(o=this._$AP)||void 0===o||o.call(this,!1,!0,e);t&&t!==this._$AB;){const e=t.nextSibling;t.remove(),t=e}}setConnected(t){var e;void 0===this._$AM&&(this._$Cm=t,null===(e=this._$AP)||void 0===e||e.call(this,t))}}class it{constructor(t,e,o,r,i){this.type=1,this._$AH=J,this._$AN=void 0,this.element=t,this.name=e,this._$AM=r,this.options=i,o.length>2||""!==o[0]||""!==o[1]?(this._$AH=Array(o.length-1).fill(new String),this.strings=o):this._$AH=J}get tagName(){return this.element.tagName}get _$AU(){return this._$AM._$AU}_$AI(t,e=this,o,r){const i=this.strings;let n=!1;if(void 0===i)t=et(this,t,e,0),n=!M(t)||t!==this._$AH&&t!==Z,n&&(this._$AH=t);else{const r=t;let a,l;for(t=i[0],a=0;a<i.length-1;a++)l=et(this,r[o+a],e,a),l===Z&&(l=this._$AH[a]),n||(n=!M(l)||l!==this._$AH[a]),l===J?t=J:t!==J&&(t+=(null!=l?l:"")+i[a+1]),this._$AH[a]=l}n&&!r&&this.j(t)}j(t){t===J?this.element.removeAttribute(this.name):this.element.setAttribute(this.name,null!=t?t:"")}}class nt extends it{constructor(){super(...arguments),this.type=3}j(t){this.element[this.name]=t===J?void 0:t}}const at=O?O.emptyScript:"";class lt extends it{constructor(){super(...arguments),this.type=4}j(t){t&&t!==J?this.element.setAttribute(this.name,at):this.element.removeAttribute(this.name)}}class st extends it{constructor(t,e,o,r,i){super(t,e,o,r,i),this.type=5}_$AI(t,e=this){var o;if((t=null!==(o=et(this,t,e,0))&&void 0!==o?o:J)===Z)return;const r=this._$AH,i=t===J&&r!==J||t.capture!==r.capture||t.once!==r.once||t.passive!==r.passive,n=t!==J&&(r===J||i);i&&this.element.removeEventListener(this.name,this,r),n&&this.element.addEventListener(this.name,this,t),this._$AH=t}handleEvent(t){var e,o;"function"==typeof this._$AH?this._$AH.call(null!==(o=null===(e=this.options)||void 0===e?void 0:e.host)&&void 0!==o?o:this.element,t):this._$AH.handleEvent(t)}}class ct{constructor(t,e,o){this.element=t,this.type=6,this._$AN=void 0,this._$AM=e,this.options=o}get _$AU(){return this._$AM._$AU}_$AI(t){et(this,t)}}const dt=z.litHtmlPolyfillSupport;null==dt||dt(tt,rt),(null!==(N=z.litHtmlVersions)&&void 0!==N?N:z.litHtmlVersions=[]).push("2.4.0");const ht=window.ShadowRoot&&(void 0===window.ShadyCSS||window.ShadyCSS.nativeShadow)&&"adoptedStyleSheets"in Document.prototype&&"replace"in CSSStyleSheet.prototype,pt=Symbol(),ut=new Map;class vt{constructor(t,e){if(this._$cssResult$=!0,e!==pt)throw Error("CSSResult is not constructable. Use `unsafeCSS` or `css` instead.");this.cssText=t}get styleSheet(){let t=ut.get(this.cssText);return ht&&void 0===t&&(ut.set(this.cssText,t=new CSSStyleSheet),t.replaceSync(this.cssText)),t}toString(){return this.cssText}}const mt=(t,...e)=>{const o=1===t.length?t[0]:e.reduce(((e,o,r)=>e+(t=>{if(!0===t._$cssResult$)return t.cssText;if("number"==typeof t)return t;throw Error("Value passed to 'css' function must be a 'css' function result: "+t+". Use 'unsafeCSS' to pass non-literal values, but take care to ensure page security.")})(o)+t[r+1]),t[0]);return new vt(o,pt)},ft=ht?t=>t:t=>t instanceof CSSStyleSheet?(t=>{let e="";for(const o of t.cssRules)e+=o.cssText;return(t=>new vt("string"==typeof t?t:t+"",pt))(e)})(t):t;var gt;const bt=window.trustedTypes,yt=bt?bt.emptyScript:"",$t=window.reactiveElementPolyfillSupport,At={toAttribute(t,e){switch(e){case Boolean:t=t?yt:null;break;case Object:case Array:t=null==t?t:JSON.stringify(t)}return t},fromAttribute(t,e){let o=t;switch(e){case Boolean:o=null!==t;break;case Number:o=null===t?null:Number(t);break;case Object:case Array:try{o=JSON.parse(t)}catch(t){o=null}}return o}},wt=(t,e)=>e!==t&&(e==e||t==t),_t={attribute:!0,type:String,converter:At,reflect:!1,hasChanged:wt};class xt extends HTMLElement{constructor(){super(),this._$Et=new Map,this.isUpdatePending=!1,this.hasUpdated=!1,this._$Ei=null,this.o()}static addInitializer(t){var e;null!==(e=this.l)&&void 0!==e||(this.l=[]),this.l.push(t)}static get observedAttributes(){this.finalize();const t=[];return this.elementProperties.forEach(((e,o)=>{const r=this._$Eh(o,e);void 0!==r&&(this._$Eu.set(r,o),t.push(r))})),t}static createProperty(t,e=_t){if(e.state&&(e.attribute=!1),this.finalize(),this.elementProperties.set(t,e),!e.noAccessor&&!this.prototype.hasOwnProperty(t)){const o="symbol"==typeof t?Symbol():"__"+t,r=this.getPropertyDescriptor(t,o,e);void 0!==r&&Object.defineProperty(this.prototype,t,r)}}static getPropertyDescriptor(t,e,o){return{get(){return this[e]},set(r){const i=this[t];this[e]=r,this.requestUpdate(t,i,o)},configurable:!0,enumerable:!0}}static getPropertyOptions(t){return this.elementProperties.get(t)||_t}static finalize(){if(this.hasOwnProperty("finalized"))return!1;this.finalized=!0;const t=Object.getPrototypeOf(this);if(t.finalize(),this.elementProperties=new Map(t.elementProperties),this._$Eu=new Map,this.hasOwnProperty("properties")){const t=this.properties,e=[...Object.getOwnPropertyNames(t),...Object.getOwnPropertySymbols(t)];for(const o of e)this.createProperty(o,t[o])}return this.elementStyles=this.finalizeStyles(this.styles),!0}static finalizeStyles(t){const e=[];if(Array.isArray(t)){const o=new Set(t.flat(1/0).reverse());for(const t of o)e.unshift(ft(t))}else void 0!==t&&e.push(ft(t));return e}static _$Eh(t,e){const o=e.attribute;return!1===o?void 0:"string"==typeof o?o:"string"==typeof t?t.toLowerCase():void 0}o(){var t;this._$Ep=new Promise((t=>this.enableUpdating=t)),this._$AL=new Map,this._$Em(),this.requestUpdate(),null===(t=this.constructor.l)||void 0===t||t.forEach((t=>t(this)))}addController(t){var e,o;(null!==(e=this._$Eg)&&void 0!==e?e:this._$Eg=[]).push(t),void 0!==this.renderRoot&&this.isConnected&&(null===(o=t.hostConnected)||void 0===o||o.call(t))}removeController(t){var e;null===(e=this._$Eg)||void 0===e||e.splice(this._$Eg.indexOf(t)>>>0,1)}_$Em(){this.constructor.elementProperties.forEach(((t,e)=>{this.hasOwnProperty(e)&&(this._$Et.set(e,this[e]),delete this[e])}))}createRenderRoot(){var t;const e=null!==(t=this.shadowRoot)&&void 0!==t?t:this.attachShadow(this.constructor.shadowRootOptions);return((t,e)=>{ht?t.adoptedStyleSheets=e.map((t=>t instanceof CSSStyleSheet?t:t.styleSheet)):e.forEach((e=>{const o=document.createElement("style"),r=window.litNonce;void 0!==r&&o.setAttribute("nonce",r),o.textContent=e.cssText,t.appendChild(o)}))})(e,this.constructor.elementStyles),e}connectedCallback(){var t;void 0===this.renderRoot&&(this.renderRoot=this.createRenderRoot()),this.enableUpdating(!0),null===(t=this._$Eg)||void 0===t||t.forEach((t=>{var e;return null===(e=t.hostConnected)||void 0===e?void 0:e.call(t)}))}enableUpdating(t){}disconnectedCallback(){var t;null===(t=this._$Eg)||void 0===t||t.forEach((t=>{var e;return null===(e=t.hostDisconnected)||void 0===e?void 0:e.call(t)}))}attributeChangedCallback(t,e,o){this._$AK(t,o)}_$ES(t,e,o=_t){var r,i;const n=this.constructor._$Eh(t,o);if(void 0!==n&&!0===o.reflect){const a=(null!==(i=null===(r=o.converter)||void 0===r?void 0:r.toAttribute)&&void 0!==i?i:At.toAttribute)(e,o.type);this._$Ei=t,null==a?this.removeAttribute(n):this.setAttribute(n,a),this._$Ei=null}}_$AK(t,e){var o,r,i;const n=this.constructor,a=n._$Eu.get(t);if(void 0!==a&&this._$Ei!==a){const t=n.getPropertyOptions(a),l=t.converter,s=null!==(i=null!==(r=null===(o=l)||void 0===o?void 0:o.fromAttribute)&&void 0!==r?r:"function"==typeof l?l:null)&&void 0!==i?i:At.fromAttribute;this._$Ei=a,this[a]=s(e,t.type),this._$Ei=null}}requestUpdate(t,e,o){let r=!0;void 0!==t&&(((o=o||this.constructor.getPropertyOptions(t)).hasChanged||wt)(this[t],e)?(this._$AL.has(t)||this._$AL.set(t,e),!0===o.reflect&&this._$Ei!==t&&(void 0===this._$EC&&(this._$EC=new Map),this._$EC.set(t,o))):r=!1),!this.isUpdatePending&&r&&(this._$Ep=this._$E_())}async _$E_(){this.isUpdatePending=!0;try{await this._$Ep}catch(t){Promise.reject(t)}const t=this.scheduleUpdate();return null!=t&&await t,!this.isUpdatePending}scheduleUpdate(){return this.performUpdate()}performUpdate(){var t;if(!this.isUpdatePending)return;this.hasUpdated,this._$Et&&(this._$Et.forEach(((t,e)=>this[e]=t)),this._$Et=void 0);let e=!1;const o=this._$AL;try{e=this.shouldUpdate(o),e?(this.willUpdate(o),null===(t=this._$Eg)||void 0===t||t.forEach((t=>{var e;return null===(e=t.hostUpdate)||void 0===e?void 0:e.call(t)})),this.update(o)):this._$EU()}catch(t){throw e=!1,this._$EU(),t}e&&this._$AE(o)}willUpdate(t){}_$AE(t){var e;null===(e=this._$Eg)||void 0===e||e.forEach((t=>{var e;return null===(e=t.hostUpdated)||void 0===e?void 0:e.call(t)})),this.hasUpdated||(this.hasUpdated=!0,this.firstUpdated(t)),this.updated(t)}_$EU(){this._$AL=new Map,this.isUpdatePending=!1}get updateComplete(){return this.getUpdateComplete()}getUpdateComplete(){return this._$Ep}shouldUpdate(t){return!0}update(t){void 0!==this._$EC&&(this._$EC.forEach(((t,e)=>this._$ES(e,this[e],t))),this._$EC=void 0),this._$EU()}updated(t){}firstUpdated(t){}}var St;xt.finalized=!0,xt.elementProperties=new Map,xt.elementStyles=[],xt.shadowRootOptions={mode:"open"},null==$t||$t({ReactiveElement:xt}),(null!==(gt=globalThis.reactiveElementVersions)&&void 0!==gt?gt:globalThis.reactiveElementVersions=[]).push("1.3.2");const kt=globalThis.trustedTypes,Et=kt?kt.createPolicy("lit-html",{createHTML:t=>t}):void 0,Ct=`lit$${(Math.random()+"").slice(9)}$`,Pt="?"+Ct,Nt=`<${Pt}>`,zt=document,Ot=(t="")=>zt.createComment(t),Rt=t=>null===t||"object"!=typeof t&&"function"!=typeof t,Ut=Array.isArray,Tt=/<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g,jt=/-->/g,Ht=/>/g,Lt=/>|[ 	\n\r](?:([^\s"'>=/]+)([ 	\n\r]*=[ 	\n\r]*(?:[^ 	\n\r"'`<>=]|("|')|))|$)/g,Mt=/'/g,It=/"/g,Dt=/^(?:script|style|textarea|title)$/i,qt=t=>(e,...o)=>({_$litType$:t,strings:e,values:o}),Bt=qt(1),Vt=(qt(2),Symbol.for("lit-noChange")),Ft=Symbol.for("lit-nothing"),Gt=new WeakMap,Wt=zt.createTreeWalker(zt,129,null,!1);class Kt{constructor({strings:t,_$litType$:e},o){let r;this.parts=[];let i=0,n=0;const a=t.length-1,l=this.parts,[s,c]=((t,e)=>{const o=t.length-1,r=[];let i,n=2===e?"<svg>":"",a=Tt;for(let e=0;e<o;e++){const o=t[e];let l,s,c=-1,d=0;for(;d<o.length&&(a.lastIndex=d,s=a.exec(o),null!==s);)d=a.lastIndex,a===Tt?"!--"===s[1]?a=jt:void 0!==s[1]?a=Ht:void 0!==s[2]?(Dt.test(s[2])&&(i=RegExp("</"+s[2],"g")),a=Lt):void 0!==s[3]&&(a=Lt):a===Lt?">"===s[0]?(a=null!=i?i:Tt,c=-1):void 0===s[1]?c=-2:(c=a.lastIndex-s[2].length,l=s[1],a=void 0===s[3]?Lt:'"'===s[3]?It:Mt):a===It||a===Mt?a=Lt:a===jt||a===Ht?a=Tt:(a=Lt,i=void 0);const h=a===Lt&&t[e+1].startsWith("/>")?" ":"";n+=a===Tt?o+Nt:c>=0?(r.push(l),o.slice(0,c)+"$lit$"+o.slice(c)+Ct+h):o+Ct+(-2===c?(r.push(void 0),e):h)}const l=n+(t[o]||"<?>")+(2===e?"</svg>":"");if(!Array.isArray(t)||!t.hasOwnProperty("raw"))throw Error("invalid template strings array");return[void 0!==Et?Et.createHTML(l):l,r]})(t,e);if(this.el=Kt.createElement(s,o),Wt.currentNode=this.el.content,2===e){const t=this.el.content,e=t.firstChild;e.remove(),t.append(...e.childNodes)}for(;null!==(r=Wt.nextNode())&&l.length<a;){if(1===r.nodeType){if(r.hasAttributes()){const t=[];for(const e of r.getAttributeNames())if(e.endsWith("$lit$")||e.startsWith(Ct)){const o=c[n++];if(t.push(e),void 0!==o){const t=r.getAttribute(o.toLowerCase()+"$lit$").split(Ct),e=/([.?@])?(.*)/.exec(o);l.push({type:1,index:i,name:e[2],strings:t,ctor:"."===e[1]?Yt:"?"===e[1]?ee:"@"===e[1]?oe:Qt})}else l.push({type:6,index:i})}for(const e of t)r.removeAttribute(e)}if(Dt.test(r.tagName)){const t=r.textContent.split(Ct),e=t.length-1;if(e>0){r.textContent=kt?kt.emptyScript:"";for(let o=0;o<e;o++)r.append(t[o],Ot()),Wt.nextNode(),l.push({type:2,index:++i});r.append(t[e],Ot())}}}else if(8===r.nodeType)if(r.data===Pt)l.push({type:2,index:i});else{let t=-1;for(;-1!==(t=r.data.indexOf(Ct,t+1));)l.push({type:7,index:i}),t+=Ct.length-1}i++}}static createElement(t,e){const o=zt.createElement("template");return o.innerHTML=t,o}}function Zt(t,e,o=t,r){var i,n,a,l;if(e===Vt)return e;let s=void 0!==r?null===(i=o._$Cl)||void 0===i?void 0:i[r]:o._$Cu;const c=Rt(e)?void 0:e._$litDirective$;return(null==s?void 0:s.constructor)!==c&&(null===(n=null==s?void 0:s._$AO)||void 0===n||n.call(s,!1),void 0===c?s=void 0:(s=new c(t),s._$AT(t,o,r)),void 0!==r?(null!==(a=(l=o)._$Cl)&&void 0!==a?a:l._$Cl=[])[r]=s:o._$Cu=s),void 0!==s&&(e=Zt(t,s._$AS(t,e.values),s,r)),e}class Jt{constructor(t,e){this.v=[],this._$AN=void 0,this._$AD=t,this._$AM=e}get parentNode(){return this._$AM.parentNode}get _$AU(){return this._$AM._$AU}p(t){var e;const{el:{content:o},parts:r}=this._$AD,i=(null!==(e=null==t?void 0:t.creationScope)&&void 0!==e?e:zt).importNode(o,!0);Wt.currentNode=i;let n=Wt.nextNode(),a=0,l=0,s=r[0];for(;void 0!==s;){if(a===s.index){let e;2===s.type?e=new Xt(n,n.nextSibling,this,t):1===s.type?e=new s.ctor(n,s.name,s.strings,this,t):6===s.type&&(e=new re(n,this,t)),this.v.push(e),s=r[++l]}a!==(null==s?void 0:s.index)&&(n=Wt.nextNode(),a++)}return i}m(t){let e=0;for(const o of this.v)void 0!==o&&(void 0!==o.strings?(o._$AI(t,o,e),e+=o.strings.length-2):o._$AI(t[e])),e++}}class Xt{constructor(t,e,o,r){var i;this.type=2,this._$AH=Ft,this._$AN=void 0,this._$AA=t,this._$AB=e,this._$AM=o,this.options=r,this._$Cg=null===(i=null==r?void 0:r.isConnected)||void 0===i||i}get _$AU(){var t,e;return null!==(e=null===(t=this._$AM)||void 0===t?void 0:t._$AU)&&void 0!==e?e:this._$Cg}get parentNode(){let t=this._$AA.parentNode;const e=this._$AM;return void 0!==e&&11===t.nodeType&&(t=e.parentNode),t}get startNode(){return this._$AA}get endNode(){return this._$AB}_$AI(t,e=this){t=Zt(this,t,e),Rt(t)?t===Ft||null==t||""===t?(this._$AH!==Ft&&this._$AR(),this._$AH=Ft):t!==this._$AH&&t!==Vt&&this.$(t):void 0!==t._$litType$?this.T(t):void 0!==t.nodeType?this.k(t):(t=>{var e;return Ut(t)||"function"==typeof(null===(e=t)||void 0===e?void 0:e[Symbol.iterator])})(t)?this.S(t):this.$(t)}M(t,e=this._$AB){return this._$AA.parentNode.insertBefore(t,e)}k(t){this._$AH!==t&&(this._$AR(),this._$AH=this.M(t))}$(t){this._$AH!==Ft&&Rt(this._$AH)?this._$AA.nextSibling.data=t:this.k(zt.createTextNode(t)),this._$AH=t}T(t){var e;const{values:o,_$litType$:r}=t,i="number"==typeof r?this._$AC(t):(void 0===r.el&&(r.el=Kt.createElement(r.h,this.options)),r);if((null===(e=this._$AH)||void 0===e?void 0:e._$AD)===i)this._$AH.m(o);else{const t=new Jt(i,this),e=t.p(this.options);t.m(o),this.k(e),this._$AH=t}}_$AC(t){let e=Gt.get(t.strings);return void 0===e&&Gt.set(t.strings,e=new Kt(t)),e}S(t){Ut(this._$AH)||(this._$AH=[],this._$AR());const e=this._$AH;let o,r=0;for(const i of t)r===e.length?e.push(o=new Xt(this.M(Ot()),this.M(Ot()),this,this.options)):o=e[r],o._$AI(i),r++;r<e.length&&(this._$AR(o&&o._$AB.nextSibling,r),e.length=r)}_$AR(t=this._$AA.nextSibling,e){var o;for(null===(o=this._$AP)||void 0===o||o.call(this,!1,!0,e);t&&t!==this._$AB;){const e=t.nextSibling;t.remove(),t=e}}setConnected(t){var e;void 0===this._$AM&&(this._$Cg=t,null===(e=this._$AP)||void 0===e||e.call(this,t))}}class Qt{constructor(t,e,o,r,i){this.type=1,this._$AH=Ft,this._$AN=void 0,this.element=t,this.name=e,this._$AM=r,this.options=i,o.length>2||""!==o[0]||""!==o[1]?(this._$AH=Array(o.length-1).fill(new String),this.strings=o):this._$AH=Ft}get tagName(){return this.element.tagName}get _$AU(){return this._$AM._$AU}_$AI(t,e=this,o,r){const i=this.strings;let n=!1;if(void 0===i)t=Zt(this,t,e,0),n=!Rt(t)||t!==this._$AH&&t!==Vt,n&&(this._$AH=t);else{const r=t;let a,l;for(t=i[0],a=0;a<i.length-1;a++)l=Zt(this,r[o+a],e,a),l===Vt&&(l=this._$AH[a]),n||(n=!Rt(l)||l!==this._$AH[a]),l===Ft?t=Ft:t!==Ft&&(t+=(null!=l?l:"")+i[a+1]),this._$AH[a]=l}n&&!r&&this.C(t)}C(t){t===Ft?this.element.removeAttribute(this.name):this.element.setAttribute(this.name,null!=t?t:"")}}class Yt extends Qt{constructor(){super(...arguments),this.type=3}C(t){this.element[this.name]=t===Ft?void 0:t}}const te=kt?kt.emptyScript:"";class ee extends Qt{constructor(){super(...arguments),this.type=4}C(t){t&&t!==Ft?this.element.setAttribute(this.name,te):this.element.removeAttribute(this.name)}}class oe extends Qt{constructor(t,e,o,r,i){super(t,e,o,r,i),this.type=5}_$AI(t,e=this){var o;if((t=null!==(o=Zt(this,t,e,0))&&void 0!==o?o:Ft)===Vt)return;const r=this._$AH,i=t===Ft&&r!==Ft||t.capture!==r.capture||t.once!==r.once||t.passive!==r.passive,n=t!==Ft&&(r===Ft||i);i&&this.element.removeEventListener(this.name,this,r),n&&this.element.addEventListener(this.name,this,t),this._$AH=t}handleEvent(t){var e,o;"function"==typeof this._$AH?this._$AH.call(null!==(o=null===(e=this.options)||void 0===e?void 0:e.host)&&void 0!==o?o:this.element,t):this._$AH.handleEvent(t)}}class re{constructor(t,e,o){this.element=t,this.type=6,this._$AN=void 0,this._$AM=e,this.options=o}get _$AU(){return this._$AM._$AU}_$AI(t){Zt(this,t)}}const ie=window.litHtmlPolyfillSupport;var ne,ae;null==ie||ie(Kt,Xt),(null!==(St=globalThis.litHtmlVersions)&&void 0!==St?St:globalThis.litHtmlVersions=[]).push("2.2.3");class le extends xt{constructor(){super(...arguments),this.renderOptions={host:this},this._$Dt=void 0}createRenderRoot(){var t,e;const o=super.createRenderRoot();return null!==(t=(e=this.renderOptions).renderBefore)&&void 0!==t||(e.renderBefore=o.firstChild),o}update(t){const e=this.render();this.hasUpdated||(this.renderOptions.isConnected=this.isConnected),super.update(t),this._$Dt=((t,e,o)=>{var r,i;const n=null!==(r=null==o?void 0:o.renderBefore)&&void 0!==r?r:e;let a=n._$litPart$;if(void 0===a){const t=null!==(i=null==o?void 0:o.renderBefore)&&void 0!==i?i:null;n._$litPart$=a=new Xt(e.insertBefore(Ot(),t),t,void 0,null!=o?o:{})}return a._$AI(t),a})(e,this.renderRoot,this.renderOptions)}connectedCallback(){var t;super.connectedCallback(),null===(t=this._$Dt)||void 0===t||t.setConnected(!0)}disconnectedCallback(){var t;super.disconnectedCallback(),null===(t=this._$Dt)||void 0===t||t.setConnected(!1)}render(){return Vt}}le.finalized=!0,le._$litElement$=!0,null===(ne=globalThis.litElementHydrateSupport)||void 0===ne||ne.call(globalThis,{LitElement:le});const se=globalThis.litElementPolyfillSupport;null==se||se({LitElement:le}),(null!==(ae=globalThis.litElementVersions)&&void 0!==ae?ae:globalThis.litElementVersions=[]).push("3.2.0");const ce=t=>e=>"function"==typeof e?((t,e)=>(customElements.define(t,e),e))(t,e):((t,e)=>{const{kind:o,elements:r}=e;return{kind:o,elements:r,finisher(e){customElements.define(t,e)}}})(t,e),de=(t,e)=>"method"===e.kind&&e.descriptor&&!("value"in e.descriptor)?{...e,finisher(o){o.createProperty(e.key,t)}}:{kind:"field",key:Symbol(),placement:"own",descriptor:{},originalKey:e.key,initializer(){"function"==typeof e.initializer&&(this[e.key]=e.initializer.call(this))},finisher(o){o.createProperty(e.key,t)}};function he(t){return(e,o)=>void 0!==o?((t,e,o)=>{e.constructor.createProperty(o,t)})(t,e,o):de(t,e)}var pe;null===(pe=window.HTMLSlotElement)||void 0===pe||pe.prototype.assignedElements;class ue extends le{get _slottedChildren(){const t=this.shadowRoot.querySelector("slot");if(t)return t.assignedElements({flatten:!0})}}const ve="categoryActivated",me=mt`
  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    padding-left: 0;
  }

  @media only screen and (max-width: 600px) {
    ul {
      margin-block-start: 0;
      margin-block-end: 0;
      margin-inline-start: 0;
      margin-inline-end: 0;
      padding-inline-start: 0;
    }
  }
`;var fe=function(t,e,o,r){var i,n=arguments.length,a=n<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)a=Reflect.decorate(t,e,o,r);else for(var l=t.length-1;l>=0;l--)(i=t[l])&&(a=(n<3?i(a):n>3?i(e,o,a):i(e,o))||a);return n>3&&a&&Object.defineProperty(e,o,a),a};let ge=class extends ue{render(){return Bt`
      <ul @categoryActivated=${this._categoryActivatedListener}>
        <slot></slot>
      </ul>
    `}firstUpdated(){setTimeout((()=>{const t=new CustomEvent(ve,{bubbles:!0,composed:!0,detail:{id:this.default,description:"All the categories, for those who like a party."}});this.dispatchEvent(t),this._categoryActivatedListener(t)}))}_categoryActivatedListener(t){for(let e=0;e<this._slottedChildren.length;e++){const o=this._slottedChildren[e];o.name!=t.detail.id?o.disableCategory():o.active||o.enableCategory()}}};ge.styles=me,fe([he()],ge.prototype,"default",void 0),ge=fe([ce("rule-category-navigation")],ge);const be=mt`
  li {
    padding-left: 0;
  }

  .active {
    background-color: var(--primary-color);
    color: var(--invert-font-color);
    font-weight: bold;
  }

  a {
    color: var(--primary-color);
    text-decoration: none;
  }

  a:hover {
    background-color: var(--primary-color);
    color: var(--invert-font-color);
  }

  @media only screen and (max-width: 600px) {
    a {
      font-size: 0.7rem;
    }
    li {
      padding-bottom: 0;
      margin-bottom: 0;
    }
  }
`;var ye=function(t,e,o,r){var i,n=arguments.length,a=n<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)a=Reflect.decorate(t,e,o,r);else for(var l=t.length-1;l>=0;l--)(i=t[l])&&(a=(n<3?i(a):n>3?i(e,o,a):i(e,o))||a);return n>3&&a&&Object.defineProperty(e,o,a),a};let $e=class extends ue{disableCategory(){this.active=!1,this.requestUpdate()}enableCategory(){this.active=!0,this.requestUpdate()}toggleCategory(t=!0){if(this.active=!this.active,t){const t={detail:{id:this.name,description:this.description},bubbles:!0,composed:!0};this.dispatchEvent(new CustomEvent(ve,t))}this.requestUpdate()}render(){return Bt`
      <li>
        <a
          href="#"
          class="${this.active?"active":""}"
          @click=${this.toggleCategory}
        >
          <slot></slot>
        </a>
      </li>
    `}};$e.styles=be,ye([he({type:String})],$e.prototype,"name",void 0),ye([he({type:Boolean})],$e.prototype,"default",void 0),ye([he({type:String})],$e.prototype,"description",void 0),$e=ye([ce("rule-category-link")],$e);let Ae=class extends ue{static get styles(){return[mt`
      .html-report {
        height: 100%;
      }
    `]}render(){return Bt`
      <div
        class="html-report"
        @categoryActivated=${this._categoryActivatedListener}
        @violationSelected=${this._violationSelectedListener}
      >
        <slot name="navigation"></slot>
        <slot name="reports"></slot>
      </div>
    `}_categoryActivatedListener(t){const e=document.querySelectorAll("category-report"),o=document.querySelectorAll("category-rule"),r=document.querySelectorAll("category-rules"),i=document.querySelector("violation-drawer"),n=this.shadowRoot.querySelector("slot").assignedElements({flatten:!0})[0].querySelector("nav").querySelector("#category-description");n&&(n.innerHTML=t.detail.description),e.forEach((e=>{e.id==t.detail.id?e.style.display="block":e.style.display="none"})),o.forEach((t=>{t.otherRuleSelected()})),r.forEach((e=>{e.id==t.detail.id&&e.rules&&e.rules.length<=0&&(e.isEmpty=!0)})),i&&i.hide()}_violationSelectedListener(){document.querySelector("violation-drawer").show()}};Ae=function(t,e,o,r){var i,n=arguments.length,a=n<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)a=Reflect.decorate(t,e,o,r);else for(var l=t.length-1;l>=0;l--)(i=t[l])&&(a=(n<3?i(a):n>3?i(e,o,a):i(e,o))||a);return n>3&&a&&Object.defineProperty(e,o,a),a}([ce("html-report")],Ae);var we=function(t,e,o,r){var i,n=arguments.length,a=n<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)a=Reflect.decorate(t,e,o,r);else for(var l=t.length-1;l>=0;l--)(i=t[l])&&(a=(n<3?i(a):n>3?i(e,o,a):i(e,o))||a);return n>3&&a&&Object.defineProperty(e,o,a),a};let _e=class extends ue{get results(){return this.shadowRoot.querySelector("slot").assignedElements({flatten:!0})}render(){return Bt`<slot></slot>`}};we([he()],_e.prototype,"id",void 0),_e=we([ce("category-report")],_e);const xe=mt`
  ul {
    margin-top: 0;
  }

  .line {
    text-align: center;
    border-radius: 5px;
    min-width: 35px;
    max-width: 35px;
    background-color: var(--card-bgcolor);
    color: var(--tertiary-color);
    font-size: var(--sl-font-size-xx-small);
  }

  .violation {
    display: flex;

    border-top: 1px solid var(--card-bgcolor);
    border-bottom: 1px solid var(--card-bgcolor);
    font-size: var(--sl-font-size-x-small);
    color: var(--font-color);
  }

  .violation:hover {
    background-color: var(--secondary-color-x-lowalpha);
    cursor: pointer;
  }

  .violation.selected:hover {
    background-color: var(--secondary-color-lowalpha);
  }

  .code-render {
    display: none;
  }

  .message {
    margin-left: 5px;
  }

  .selected {
    background-color: var(--secondary-color-lowalpha);
  }

  .selected .line {
    color: var(--font-color);
  }

  .selected .message {
    font-weight: bold;
  }
`;var Se=function(t,e,o,r){var i,n=arguments.length,a=n<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)a=Reflect.decorate(t,e,o,r);else for(var l=t.length-1;l>=0;l--)(i=t[l])&&(a=(n<3?i(a):n>3?i(e,o,a):i(e,o))||a);return n>3&&a&&Object.defineProperty(e,o,a),a};let ke=class extends ue{connectedCallback(){super.connectedCallback(),this._violationId=Math.random().toString(20).substring(2)}get violationId(){return this._violationId}render(){return Bt` <nav
        aria-label="Violation Navigation"
        class="violation ${this.selected?"selected":""}"
        @click=${this._violationClicked}
      >
        <div class="line">${this.startLine}</div>
        <div class="message">${this.path}</div>
      </nav>
      <div class="code-render">
        <slot></slot>
      </div>`}_violationClicked(){let t;this._renderedCode?t=this._renderedCode:(t=this._slottedChildren[0],this._renderedCode=t);const e={detail:{message:this.message,id:this.ruleId,startLine:this.startLine,startCol:this.startCol,endLine:this.endLine,endCol:this.endCol,path:this.path,category:this.category,howToFix:this.howToFix,violationId:this.violationId,renderedCode:t},bubbles:!0,composed:!0};this.dispatchEvent(new CustomEvent("violationSelected",e))}};ke.styles=xe,Se([he({type:String})],ke.prototype,"message",void 0),Se([he({type:String})],ke.prototype,"category",void 0),Se([he({type:String})],ke.prototype,"ruleId",void 0),Se([he({type:Number})],ke.prototype,"startLine",void 0),Se([he({type:Number})],ke.prototype,"startCol",void 0),Se([he({type:Number})],ke.prototype,"endLine",void 0),Se([he({type:Number})],ke.prototype,"endCol",void 0),Se([he({type:String})],ke.prototype,"path",void 0),Se([he({type:String})],ke.prototype,"howToFix",void 0),Se([he({type:Boolean})],ke.prototype,"selected",void 0),ke=Se([ce("category-rule-result")],ke);const Ee=mt`
  .rule-icon {
    font-family: 'Arial';
    font-size: var(--sl-font-size-small);
    width: 20px;
    display: inline-block;
  }

  li {
    margin: 0;
    padding-left: 0;
  }

  li::after {
    content: '';
  }

  .details {
    margin-bottom: calc(var(--global-margin) / 2);
  }

  .details > .summary {
    background-color: var(--card-bgcolor);
    border: 1px solid var(--card-bordercolor);
    padding: 5px;
    border-radius: 3px;
  }

  .rule-violation-count {
    font-size: var(--sl-font-size-x-small);
    border: 1px solid var(--card-bordercolor);
    color: var(--tertiary-color);
    padding: 2px;
    border-radius: 2px;
  }

  .details.open .summary {
    background-color: var(--primary-color);
    color: var(--invert-font-color);
    font-weight: bold;
  }

  .details.open .rule-violation-count {
    background-color: var(--primary-color);
    color: var(--invert-font-color);
    border: 1px solid var(--invert-font-color);
    font-weight: normal;
  }

  .details.open .expand-state {
    color: var(--invert-font-color);
  }

  .details > div.violations {
    font-size: var(--sl-font-size-x-small);
    overflow-y: auto;
    overflow-x: hidden;
    border: 1px solid var(--card-bordercolor);
  }

  @media only screen and (max-width: 1200px) {
    .details > div.violations {
      max-height: 230px;
    }
  }

  .details > .summary::marker {
    color: var(--secondary-color);
  }

  .rule-description {
    font-size: var(--rule-font-size);
  }

  .summary:hover {
    cursor: pointer;
    background-color: var(--primary-color-lowalpha);
    color: var(--invert-font-color);
  }

  .summary:hover .expand-state {
    color: var(--invert-font-color);
  }

  .summary:hover .rule-violation-count {
    color: var(--invert-font-color);
    border: 1px solid var(--invert-font-color);
  }

  .violations {
    display: none;
    scrollbar-width: thin;
  }

  .violations::-webkit-scrollbar {
    width: 8px;
  }

  .violations::-webkit-scrollbar-track {
    background-color: var(--card-bgcolor);
  }

  .violations::-webkit-scrollbar-thumb {
    box-shadow: inset 0 0 6px rgba(0, 0, 0, 0.3);
    background: var(--primary-color-lowalpha);
  }

  .expand-state {
    color: var(--font-color);
    vertical-align: sub;
    height: 20px;
    width: 20px;
    display: inline-block;
  }

  .expand-state:hover {
    cursor: pointer;
    color: var(--primary-color);
  }

  .truncated {
    text-align: center;
    color: var(--error-color);
    border: 1px solid var(--error-color);
    padding: var(--global-padding);
    margin-bottom: 1px;
    margin-right: 1px;
  }

  @media only screen and (max-width: 600px) {
    .details {
      max-height: 300px;
      overflow-y: hidden;
    }
  }
`,Ce=Bt`
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="20"
    height="20"
    fill="currentColor"
    class="bi bi-plus-square"
    viewBox="0 0 16 16"
  >
    <path
      d="M14 1a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1h12zM2 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H2z"
    />
    <path
      d="M8 4a.5.5 0 0 1 .5.5v3h3a.5.5 0 0 1 0 1h-3v3a.5.5 0 0 1-1 0v-3h-3a.5.5 0 0 1 0-1h3v-3A.5.5 0 0 1 8 4z"
    />
  </svg>
`,Pe=Bt`
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="20"
    height="20"
    fill="currentColor"
    class="bi bi-dash-square"
    viewBox="0 0 16 16"
  >
    <path
      d="M14 1a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1h12zM2 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H2z"
    />
    <path d="M4 8a.5.5 0 0 1 .5-.5h7a.5.5 0 0 1 0 1h-7A.5.5 0 0 1 4 8z" />
  </svg>
`;var Ne=function(t,e,o,r){var i,n=arguments.length,a=n<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)a=Reflect.decorate(t,e,o,r);else for(var l=t.length-1;l>=0;l--)(i=t[l])&&(a=(n<3?i(a):n>3?i(e,o,a):i(e,o))||a);return n>3&&a&&Object.defineProperty(e,o,a),a};let ze=class extends ue{otherRuleSelected(){this.open=!1,this.violations=this.renderRoot.querySelector(".violations"),this.violations.style.display="none",this._expandState=!1,this._slottedChildren.forEach((t=>{t.selected=!1})),this.requestUpdate()}render(){let t;this.violations=this.renderRoot.querySelector(".violations"),this.truncated&&(t=Bt`
        <div class="truncated">
          <strong>${this.numResults-this.maxViolations}</strong> more
          violations not rendered, There are just too many!
        </div>
      `);const e=this._expandState?Pe:Ce;return Bt`
      <nav
        aria-label="Rules and Violations"
        class="details ${this._expandState?"open":""}"
      >
        <div class="summary" @click=${this._ruleSelected}>
          <span class="expand-state">${e}</span>
          <span class="rule-icon">${this.ruleIcon}</span>
          <span class="rule-description">${this.description}</span>
          <span class="rule-violation-count">${this.numResults}</span>
        </div>
        <div class="violations" @violationSelected=${this._violationSelected}>
          <slot name="results"></slot>
          ${t}
        </div>
      </nav>
    `}_ruleSelected(){if(this.open)this.violations.style.display="none",this._expandState=!1;else{this.violations.style.display="block";const t=this.parentElement.parentElement.offsetHeight-60*this.totalRulesViolated;this.violations.style.maxHeight=t+"px",this._expandState=!0}this.open=!this.open,this.dispatchEvent(new CustomEvent("ruleSelected",{bubbles:!0,composed:!0,detail:{id:this.ruleId}})),this.requestUpdate()}_violationSelected(t){this._slottedChildren.forEach((e=>{e.selected=t.detail.violationId==e.violationId}))}};ze.styles=Ee,Ne([he()],ze.prototype,"totalRulesViolated",void 0),Ne([he()],ze.prototype,"maxViolations",void 0),Ne([he()],ze.prototype,"truncated",void 0),Ne([he()],ze.prototype,"ruleId",void 0),Ne([he()],ze.prototype,"description",void 0),Ne([he()],ze.prototype,"numResults",void 0),Ne([he()],ze.prototype,"ruleIcon",void 0),Ne([he()],ze.prototype,"open",void 0),ze=Ne([ce("category-rule")],ze);const Oe=mt`
  ul.rule {
    margin: 0;
    padding: 0;
  }

  section {
    //max-height: 35vh;
    overflow-y: hidden;
  }

  p {
    font-size: var(--sl-font-size-small);
    margin: 0;
  }

  .symbol {
    font-family: Arial;
  }

  section.no-violations {
    border: 1px solid var(--terminal-green);
    padding: 20px;
    color: var(--terminal-green);
    text-align: center;
  }
`;var Re=function(t,e,o,r){var i,n=arguments.length,a=n<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)a=Reflect.decorate(t,e,o,r);else for(var l=t.length-1;l>=0;l--)(i=t[l])&&(a=(n<3?i(a):n>3?i(e,o,a):i(e,o))||a);return n>3&&a&&Object.defineProperty(e,o,a),a};let Ue=class extends ue{render(){return this.isEmpty?Bt`
        <section class="no-violations">
          <p>All good in here, no rules broken!</p>
        </section>
      `:Bt`
        <section @ruleSelected=${this._ruleSelected}>
          <ul class="rule">
            <slot></slot>
          </ul>
        </section>
      `}get rules(){const t=this.shadowRoot.querySelector("slot");if(t)return t.assignedElements({flatten:!0})}_ruleSelected(t){this.rules.forEach((e=>{e.ruleId!=t.detail.id&&e.otherRuleSelected()}))}};Ue.styles=Oe,Re([he()],Ue.prototype,"id",void 0),Re([he()],Ue.prototype,"isEmpty",void 0),Ue=Re([ce("category-rules")],Ue);const Te=mt`
  /* Background */

  .chroma {
    color: #f8f8f2;
    background-color: var(--code-bg-color);
  }

  /* Other */

  .chroma .x {
  }

  /* Error */

  .chroma .err {
  }

  /* LineTableTD */

  .chroma .lntd {
    vertical-align: top;
    padding: 0;
    margin: 0;
    border: 0;
  }

  /* LineTable */

  .chroma .lntable {
    border-spacing: 0;
    padding: 0;
    margin: 0;
    border: 0;
    width: auto;
    overflow: auto;
    display: block;
  }

  /* LineHighlight */

  .chroma .hl {
    display: block;
    width: 100%;
    background-color: rgba(98, 196, 255, 0.12);
  }
  
  .hl > span.ln {
    color: white;
  }
  
  /* LineNumbersTable */

  .chroma .lnt {
    margin-right: 0.4em;
    padding: 0 0.4em 0 0.4em;
    color: #7f7f7f;
  }

  /* LineNumbers */

  .chroma .ln {
    margin-right: 0.4em;
    padding: 0 0.4em 0 0.4em;
    color: #7f7f7f;
  }

  /* Keyword */

  .chroma .k {
    color: #b584fd;
  }

  /* KeywordConstant */

  .chroma .kc {
    color: #c8a1fd;
  }

  /* KeywordDeclaration */

  .chroma .kd {
    color: #8be9fd;
    font-style: italic;
  }

  /* KeywordNamespace */

  .chroma .kn {
    color: #ff79c6;
  }

  /* KeywordPseudo */

  .chroma .kp {
    color: #ff79c6;
  }

  /* KeywordReserved */

  .chroma .kr {
    color: #ff79c6;
  }

  /* KeywordType */

  .chroma .kt {
    color: #8be9fd;
  }

  /* Name */

  .chroma .n {
  }

  /* NameAttribute */

  .chroma .na {
    color: #50fa7b;
  }

  /* NameBuiltin */

  .chroma .nb {
    color: #8be9fd;
    font-style: italic;
  }

  /* NameBuiltinPseudo */

  .chroma .bp {
  }

  /* NameClass */

  .chroma .nc {
    color: #50fa7b;
  }

  /* NameConstant */

  .chroma .no {
  }

  /* NameDecorator */

  .chroma .nd {
  }

  /* NameEntity */

  .chroma .ni {
  }

  /* NameException */

  .chroma .ne {
  }

  /* NameFunction */

  .chroma .nf {
    color: #50fa7b;
  }

  /* NameFunctionMagic */

  .chroma .fm {
  }

  /* NameLabel */

  .chroma .nl {
    color: #8be9fd;
    font-style: italic;
  }

  /* NameNamespace */

  .chroma .nn {
  }

  /* NameOther */

  .chroma .nx {
  }

  /* NameProperty */

  .chroma .py {
  }

  /* NameTag */

  .chroma .nt {
    color: #b584fd;
  }

  /* NameVariable */

  .chroma .nv {
    color: #8be9fd;
    font-style: italic;
  }

  /* NameVariableClass */

  .chroma .vc {
    color: #8be9fd;
    font-style: italic;
  }

  /* NameVariableGlobal */

  .chroma .vg {
    color: #8be9fd;
    font-style: italic;
  }

  /* NameVariableInstance */

  .chroma .vi {
    color: #8be9fd;
    font-style: italic;
  }

  /* NameVariableMagic */

  .chroma .vm {
  }

  /* Literal */

  .chroma .l {
    color: var(--primary-color);
  }
  }

  /* LiteralDate */

  .chroma .ld {
  }

  /* LiteralString */

  .chroma .s {
    color: #717684;
  }

  /* LiteralStringAffix */

  .chroma .sa {
    color: #717684;
  }

  /* LiteralStringBacktick */

  .chroma .sb {
    color: #717684;
  }

  /* LiteralStringChar */

  .chroma .sc {
    color: #717684;
  }

  /* LiteralStringDelimiter */

  .chroma .dl {
    color: #717684;
  }

  /* LiteralStringDoc */

  .chroma .sd {
    color: #717684;
  }

  /* LiteralStringDouble */

  .chroma .s2 {
    color: var(--primary-color);
  }

  /* LiteralStringEscape */

  .chroma .se {
    color: #717684;
  }

  /* LiteralStringHeredoc */

  .chroma .sh {
    color: #717684;
  }

  /* LiteralStringInterpol */

  .chroma .si {
    color: #717684;
  }

  /* LiteralStringOther */

  .chroma .sx {
    color: #717684;
  }

  /* LiteralStringRegex */

  .chroma .sr {
    color: #717684;

    /* LiteralStringSingle */

    .chroma .s1 {
      color: var(--primary-color);
    }

    /* LiteralStringSymbol */

    .chroma .ss {
      color: #717684;
    }

    /* LiteralNumber */

    .chroma .m {
      color: #bd93f9;
    }

    /* LiteralNumberBin */

    .chroma .mb {
      color: #bd93f9;
    }

    /* LiteralNumberFloat */

    .chroma .mf {
      color: #bd93f9;
    }

    /* LiteralNumberHex */

    .chroma .mh {
      color: #bd93f9;
    }

    /* LiteralNumberInteger */

    .chroma .mi {
      color: #bd93f9;
    }

    /* LiteralNumberIntegerLong */

    .chroma .il {
      color: #bd93f9;
    }

    /* LiteralNumberOct */

    .chroma .mo {
      color: #bd93f9;
    }

    /* Operator */

    .chroma .o {
      color: #ff79c6;
    }

    /* OperatorWord */

    .chroma .ow {
      color: #ff79c6;
    }

    /* Punctuation */

    .chroma .p {
    }

    /* Comment */

    .chroma .c {
      color: #6272a4;
    }

    /* CommentHashbang */

    .chroma .ch {
      color: #6272a4;
    }

    /* CommentMultiline */

    .chroma .cm {
      color: #6272a4;
    }

    /* CommentSingle */

    .chroma .c1 {
      color: #6272a4;
    }

    /* CommentSpecial */

    .chroma .cs {
      color: #6272a4;
    }

    /* CommentPreproc */

    .chroma .cp {
      color: #ff79c6;
    }

    /* CommentPreprocFile */

    .chroma .cpf {
      color: #ff79c6;
    }

    /* Generic */

    .chroma .g {
    }

    /* GenericDeleted */

    .chroma .gd {
      color: #ff5555;
    }

    /* GenericEmph */

    .chroma .ge {
      text-decoration: underline;
    }

    /* GenericError */

    .chroma .gr {
    }

    /* GenericHeading */

    .chroma .gh {
      font-weight: bold;
    }

    /* GenericInserted */

    .chroma .gi {
      color: #50fa7b;
      font-weight: bold;
    }

    /* GenericOutput */

    .chroma .go {
      color: #44475a;
    }

    /* GenericPrompt */

    .chroma .gp {
    }

    /* GenericStrong */

    .chroma .gs {
    }

    /* GenericSubheading */

    .chroma .gu {
      font-weight: bold;
    }

    /* GenericTraceback */

    .chroma .gt {
    }

    /* GenericUnderline */

    .chroma .gl {
      text-decoration: underline;
    }

    /* TextWhitespace */

    .chroma .w {
    }
`;let je=class extends ue{static get styles(){const t=mt``;return[Te,t]}render(){return Bt`
      <slot
        @violationSelected=${this._violationSelectedListener}
        name="violation"
      ></slot>
      <slot name="details"></slot>
    `}_violationSelectedListener(t){const e=this.shadowRoot.querySelectorAll("slot")[1].assignedElements({flatten:!0})[0];e.ruleId=t.detail.id,e.message=t.detail.message,e.code=t.detail.renderedCode,e.howToFix=t.detail.howToFix,e.category=t.detail.category,e.path=t.detail.path}};je=function(t,e,o,r){var i,n=arguments.length,a=n<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)a=Reflect.decorate(t,e,o,r);else for(var l=t.length-1;l>=0;l--)(i=t[l])&&(a=(n<3?i(a):n>3?i(e,o,a):i(e,o))||a);return n>3&&a&&Object.defineProperty(e,o,a),a}([ce("result-grid")],je);const He=[Te,mt`
    hr {
      border: 0;
      border-top: 1px dashed var(--secondary-color-lowalpha);
      margin-top: var(--global-margin);
      margin-bottom: var(--global-margin);
    }

    pre {
      overflow-x: auto;
    }

    pre::-webkit-scrollbar {
      height: 8px;
    }
    pre::-webkit-scrollbar-track {
      background-color: var(--card-bgcolor);
    }

    pre::-webkit-scrollbar-thumb {
      box-shadow: inset 0 0 6px rgba(0, 0, 0, 0.3);
      background: var(--primary-color-lowalpha);
    }

    p.violated {
      font-size: var(--sl-font-size-small);
    }

    pre {
      font-size: var(--sl-font-size-x-small);
    }

    a {
      font-size: var(--sl-font-size-small);
      color: var(--primary-color);
    }
    a:hover {
      background-color: var(--secondary-color);
      cursor: pointer;
      color: var(--invert-font-color);
    }
    h2 {
      margin-top: 0;
      line-height: 2.3rem;
      font-size: 1.4rem;
    }

    .backtick-element {
      background-color: black;
      color: var(--secondary-color);
      border: 1px solid var(--secondary-color-lowalpha);
      border-radius: 5px;
      padding: 2px;
    }

    section.select-violation {
      width: 100%;
      text-align: center;
    }
    section.select-violation p {
      color: var(--secondary-color-lowalpha);
      font-size: var(--sl-font-size-x-small);
    }

    section.how-to-fix p {
      font-size: var(--sl-font-size-x-small);
    }

    p.path {
      color: var(--secondary-color);
    }

    @media only screen and (max-width: 600px) {
      h2 {
        font-size: 1rem;
      }
    }
  `];var Le,Me=function(t,e,o,r){var i,n=arguments.length,a=n<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)a=Reflect.decorate(t,e,o,r);else for(var l=t.length-1;l>=0;l--)(i=t[l])&&(a=(n<3?i(a):n>3?i(e,o,a):i(e,o))||a);return n>3&&a&&Object.defineProperty(e,o,a),a};let Ie=Le=class extends ue{static replaceTicks(t){const e=/(`[^`]*`)/g,o=t.split(e),r=new Array;return o.forEach((t=>{if(t.match(e)){const e=t.replace(/`/g,""),o=Bt`<span class="backtick-element">${e}</span>`;r.push(o)}else""!=t&&r.push(Bt`${t}`)})),r}render(){return this._visible?Bt`
        <h2>${Le.replaceTicks(this.message)}</h2>
        ${this.code}
        <h3>JSON Path</h3>
        <p class="path">${this.path}</p>
        <hr />
        <section class="how-to-fix">
          <h3>How to fix violation</h3>
          <p>${this.howToFix}</p>
        </section>
        <hr />
        <p class="violated">
          Learn more about:
          <a
            href="https://quobix.com/vacuum/rules/${this.category}/${this.ruleId}"
            >${this.ruleId}</a
          >
        </p>
      `:Bt`
        <section class="select-violation">
          <p>Please select a rule violation from a category.</p>
        </section>
      `}get drawer(){return document.querySelector("violation-drawer")}show(){this._visible=!0,this.drawer.classList.add("drawer-active"),this.requestUpdate()}hide(){this._visible=!1,this.drawer.classList.remove("drawer-active"),this.requestUpdate()}};Ie.styles=He,Me([he({type:Element})],Ie.prototype,"code",void 0),Me([he({type:String})],Ie.prototype,"message",void 0),Me([he({type:String})],Ie.prototype,"path",void 0),Me([he({type:String})],Ie.prototype,"category",void 0),Me([he({type:String})],Ie.prototype,"ruleId",void 0),Me([he({type:String})],Ie.prototype,"howToFix",void 0),Ie=Le=Me([ce("violation-drawer")],Ie);var De=function(t,e,o,r){var i,n=arguments.length,a=n<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)a=Reflect.decorate(t,e,o,r);else for(var l=t.length-1;l>=0;l--)(i=t[l])&&(a=(n<3?i(a):n>3?i(e,o,a):i(e,o))||a);return n>3&&a&&Object.defineProperty(e,o,a),a};let qe=class extends ue{static get styles(){return[mt`
      span {
        display: block;
      }

      div {
        padding: 5px;
        min-width: 80px;
      }

      span.grade {
        font-size: 1.3rem;
        font-weight: bold;
      }

      span.label {
        font-size: var(--sl-font-size-xx-small);
        color: var(--font-color);
      }

      .error {
        background-color: var(--error-color-lowalpha);
        border: 1px solid var(--error-color);
        color: var(--error-color);
      }

      .warn-400 {
        background-color: var(--warn-400-lowalpha);
        border: 1px solid var(--warn-400);
        color: var(--warn-400);
      }

      .warn-300 {
        background-color: var(--warn-300-lowalpha);
        border: 1px solid var(--warn-300);
        color: var(--warn-300);
      }

      .warn-200 {
        background-color: var(--warn-200-lowalpha);
        border: 1px solid var(--warn-200);
        color: var(--warn-200);
      }

      .warn {
        background-color: var(--warn-color-lowalpha);
        border: 1px solid var(--warn-color);
        color: var(--warn-color);
      }

      .ok-400 {
        background-color: var(--ok-400-lowalpha);
        border: 1px solid var(--ok-400);
        color: var(--ok-400);
      }

      .ok-300 {
        background-color: var(--ok-300-lowalpha);
        border: 1px solid var(--ok-300);
        color: var(--ok-300);
      }

      .ok-200 {
        background-color: var(--ok-200-lowalpha);
        border: 1px solid var(--ok-200);
        color: var(--ok-200);
      }

      .ok {
        background-color: var(--ok-color-lowalpha);
        border: 1px solid var(--ok-color);
        color: var(--ok-color);
      }

      .warning-count {
        background: none;
        color: var(--primary-color);
      }

      .error-count {
        background: none;
        color: var(--primary-color);
      }

      .info-count {
        background: none;
        color: var(--primary-color);
      }

      @media only screen and (max-width: 600px) {
        div {
          padding: 5px;
          min-width: 60px;
        }
      }
    `]}render(){return Bt`
      <div class=${this.colorForScore()}>
        <span class="grade"
          >${this.value.toLocaleString()}${this.percentage?"%":""}</span
        >
        <span class="label"> ${this.label} </span>
      </div>
    `}colorForScore(){if(this.preset)return this.preset;switch(!0){case this.value<=10:return"error";case this.value>10&&this.value<20:return"warn-400";case this.value>=20&&this.value<30:return"warn-300";case this.value>=30&&this.value<40:return"warn-200";case this.value>=40&&this.value<50:return"warn";case this.value>=50&&this.value<65:return"ok-400";case this.value>=65&&this.value<75:return"ok-300";case this.value>=75&&this.value<95:return"ok-200";case this.value>=95:default:return"ok"}}};De([he({type:Number})],qe.prototype,"value",void 0),De([he()],qe.prototype,"preset",void 0),De([he()],qe.prototype,"percentage",void 0),De([he()],qe.prototype,"label",void 0),qe=De([ce("header-statistic")],qe)})()})();
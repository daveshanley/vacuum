/*! For license information please see vacuumReport.js.LICENSE.txt */
(()=>{"use strict";var e={408:(e,t,o)=>{o.d(t,{Z:()=>l});var r=o(81),i=o.n(r),a=o(645),n=o.n(a)()(i());n.push([e.id,':root{--global-font-size:15px;--global-line-height:1.4em;--global-space:10px;--font-stack:Menlo,Monaco,Lucida Console,Liberation Mono,DejaVu Sans Mono,Bitstream Vera Sans Mono,Courier New,monospace,serif;--mono-font-stack:Menlo,Monaco,Lucida Console,Liberation Mono,DejaVu Sans Mono,Bitstream Vera Sans Mono,Courier New,monospace,serif;--background-color:#fff;--page-width:60em;--font-color:#151515;--invert-font-color:#fff;--primary-color:#1a95e0;--secondary-color:#727578;--error-color:#d20962;--progress-bar-background:#727578;--progress-bar-fill:#151515;--code-bg-color:#e8eff2;--input-style:solid;--display-h1-decoration:none}*{box-sizing:border-box;text-rendering:geometricPrecision}::-moz-selection{background:var(--primary-color);color:var(--invert-font-color)}::selection{background:var(--primary-color);color:var(--invert-font-color)}body{font-size:var(--global-font-size);color:var(--font-color);line-height:var(--global-line-height);margin:0;font-family:var(--font-stack);word-wrap:break-word;background-color:var(--background-color)}.logo,h1,h2,h3,h4,h5,h6{line-height:var(--global-line-height)}a{cursor:pointer;color:var(--primary-color);text-decoration:none}a:hover{background-color:var(--primary-color);color:var(--invert-font-color)}em{font-size:var(--global-font-size);font-style:italic;font-family:var(--font-stack);color:var(--font-color)}blockquote,code,em,strong{line-height:var(--global-line-height)}.logo,blockquote,code,footer,h1,h2,h3,h4,h5,h6,header,li,ol,p,section,ul{float:none;margin:0;padding:0}.logo,blockquote,h1,ol,p,ul{margin-top:calc(var(--global-space) * 2);margin-bottom:calc(var(--global-space) * 2)}.logo,h1{position:relative;display:inline-block;display:table-cell;padding:calc(var(--global-space) * 2) 0 calc(var(--global-space) * 2);margin:0;overflow:hidden;font-weight:600}h1::after{content:"====================================================================================================";position:absolute;bottom:5px;left:0;display:var(--display-h1-decoration)}.logo+*,h1+*{margin-top:0}h2,h3,h4,h5,h6{position:relative;margin-bottom:var(--global-line-height);font-weight:600}blockquote{position:relative;padding-left:calc(var(--global-space) * 2);padding-left:2ch;overflow:hidden}blockquote::after{content:">\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>\\A>";white-space:pre;position:absolute;top:0;left:0;line-height:var(--global-line-height);color:#9ca2ab}code{font-weight:inherit;background-color:var(--code-bg-color);font-family:var(--mono-font-stack)}code::after,code::before{content:"`";display:inline}pre code::after,pre code::before{content:""}pre{display:block;word-break:break-all;word-wrap:break-word;color:var(--secondary-color);background-color:var(--background-color);border:1px solid var(--secondary-color);padding:var(--global-space);white-space:pre-wrap;white-space:-moz-pre-wrap;white-space:-pre-wrap;white-space:-o-pre-wrap}pre code{overflow-x:scroll;padding:0;margin:0;display:inline-block;min-width:100%;font-family:var(--mono-font-stack)}.terminal .logo,.terminal blockquote,.terminal code,.terminal h1,.terminal h2,.terminal h3,.terminal h4,.terminal h5,.terminal h6,.terminal strong{font-size:var(--global-font-size);font-style:normal;font-family:var(--font-stack);color:var(--font-color)}.terminal-prompt{position:relative;white-space:nowrap}.terminal-prompt::before{content:"> "}.terminal-prompt::after{content:"";-webkit-animation:cursor .8s infinite;animation:cursor .8s infinite;background:var(--primary-color);border-radius:0;display:inline-block;height:1em;margin-left:.2em;width:3px;bottom:-2px;position:relative}@-webkit-keyframes cursor{0%{opacity:0}50%{opacity:1}to{opacity:0}}@keyframes cursor{0%{opacity:0}50%{opacity:1}to{opacity:0}}li,li>ul>li{position:relative;display:block;padding-left:calc(var(--global-space) * 2)}nav>ul>li{padding-left:0}li::after{position:absolute;top:0;left:0}ul>li::after{content:"-"}nav ul>li::after{content:""}ol li::before{content:counters(item, ".") ". ";counter-increment:item}ol ol li::before{content:counters(item, ".") " ";counter-increment:item}.terminal-menu li::after,.terminal-menu li::before{display:none}ol{counter-reset:item}ol li:nth-child(n+10)::after{left:-7px}ol ol{margin-top:0;margin-bottom:0}.terminal-menu{width:100%}.terminal-nav{display:flex;flex-direction:column;align-items:flex-start}ul ul{margin-top:0;margin-bottom:0}.terminal-menu ul{list-style-type:none;padding:0!important;display:flex;flex-direction:column;width:100%;flex-grow:1;font-size:var(--global-font-size);margin-top:0}.terminal-menu li{display:flex;margin:0 0 .5em 0;padding:0}ol.terminal-toc li{border-bottom:1px dotted var(--secondary-color);padding:0;margin-bottom:15px}.terminal-menu li:last-child{margin-bottom:0}ol.terminal-toc li a{margin:4px 4px 4px 0;background:var(--background-color);position:relative;top:6px;text-align:left;padding-right:4px}.terminal-menu li a:not(.btn){text-decoration:none;display:block;width:100%;border:none;color:var(--secondary-color)}.terminal-menu li a.active{color:var(--font-color)}.terminal-menu li a:hover{background:0 0;color:inherit}ol.terminal-toc li::before{content:counters(item, ".") ". ";counter-increment:item;position:absolute;right:0;background:var(--background-color);padding:4px 0 4px 4px;bottom:-8px}ol.terminal-toc li a:hover{background:var(--primary-color);color:var(--invert-font-color)}hr{position:relative;overflow:hidden;margin:calc(var(--global-space) * 4) 0;border:0;border-bottom:1px dashed var(--secondary-color)}p{margin:0 0 var(--global-line-height);color:var(--global-font-color)}.container{max-width:var(--page-width)}.container,.container-fluid{margin:0 auto;padding:0 calc(var(--global-space) * 2)}img{width:100%}.progress-bar{height:8px;background-color:var(--progress-bar-background);margin:12px 0}.progress-bar.progress-bar-show-percent{margin-top:38px}.progress-bar-filled{background-color:var(--progress-bar-fill);height:100%;transition:width .3s ease;position:relative;width:0}.progress-bar-filled::before{content:"";border:6px solid transparent;border-top-color:var(--progress-bar-fill);position:absolute;top:-12px;right:-6px}.progress-bar-filled::after{color:var(--progress-bar-fill);content:attr(data-filled);display:block;font-size:12px;white-space:nowrap;position:absolute;border:6px solid transparent;top:-38px;right:0;transform:translateX(50%)}.progress-bar-no-arrow>.progress-bar-filled::after,.progress-bar-no-arrow>.progress-bar-filled::before{content:"";display:none;visibility:hidden;opacity:0}table{width:100%;border-collapse:collapse;margin:var(--global-line-height) 0;color:var(--font-color);font-size:var(--global-font-size)}table td,table th{vertical-align:top;border:1px solid var(--font-color);line-height:var(--global-line-height);padding:calc(var(--global-space)/ 2);font-size:1em}table thead th{font-size:1em}table tfoot tr th{font-weight:500}table caption{font-size:1em;margin:0 0 1em 0}table tbody td:first-child{font-weight:700;color:var(--secondary-color)}.form{width:100%}fieldset{border:1px solid var(--font-color);padding:1em}label{font-size:1em;color:var(--font-color)}input[type=email],input[type=number],input[type=password],input[type=search],input[type=text]{border:1px var(--input-style) var(--font-color);width:100%;padding:.7em .5em;font-size:1em;font-family:var(--font-stack);-webkit-appearance:none;border-radius:0}input[type=email]:active,input[type=email]:focus,input[type=number]:active,input[type=number]:focus,input[type=password]:active,input[type=password]:focus,input[type=search]:active,input[type=search]:focus,input[type=text]:active,input[type=text]:focus{outline:0;-webkit-appearance:none;border:1px solid var(--font-color)}input[type=email]:not(:placeholder-shown):invalid,input[type=number]:not(:placeholder-shown):invalid,input[type=password]:not(:placeholder-shown):invalid,input[type=search]:not(:placeholder-shown):invalid,input[type=text]:not(:placeholder-shown):invalid{border-color:var(--error-color)}input,textarea{color:var(--font-color);background-color:var(--background-color)}input::-webkit-input-placeholder,textarea::-webkit-input-placeholder{color:var(--secondary-color)!important;opacity:1}input::-moz-placeholder,textarea::-moz-placeholder{color:var(--secondary-color)!important;opacity:1}input:-ms-input-placeholder,textarea:-ms-input-placeholder{color:var(--secondary-color)!important;opacity:1}input::-ms-input-placeholder,textarea::-ms-input-placeholder{color:var(--secondary-color)!important;opacity:1}input::placeholder,textarea::placeholder{color:var(--secondary-color)!important;opacity:1}textarea{height:auto;width:100%;resize:none;border:1px var(--input-style) var(--font-color);padding:.5em;font-size:1em;font-family:var(--font-stack);-webkit-appearance:none;border-radius:0}textarea:focus{outline:0;-webkit-appearance:none;border:1px solid var(--font-color)}textarea:not(:placeholder-shown):invalid{border-color:var(--error-color)}input:-webkit-autofill,input:-webkit-autofill:focus textarea:-webkit-autofill,input:-webkit-autofill:hover,select:-webkit-autofill,select:-webkit-autofill:focus,select:-webkit-autofill:hover,textarea:-webkit-autofill:hover textarea:-webkit-autofill:focus{border:1px solid var(--font-color);-webkit-text-fill-color:var(--font-color);box-shadow:0 0 0 1000px var(--invert-font-color) inset;-webkit-box-shadow:0 0 0 1000px var(--invert-font-color) inset;transition:background-color 5000s ease-in-out 0s}.form-group{margin-bottom:var(--global-line-height);overflow:auto}.btn{border-style:solid;border-width:1px;display:inline-flex;align-items:center;justify-content:center;cursor:pointer;outline:0;padding:.65em 2em;font-size:1em;font-family:inherit;-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;position:relative;z-index:1}.btn:active{box-shadow:none}.btn.btn-ghost{border-color:var(--font-color);color:var(--font-color);background-color:transparent}.btn.btn-ghost:focus,.btn.btn-ghost:hover{border-color:var(--tertiary-color);color:var(--tertiary-color);z-index:2}.btn.btn-ghost:hover{background-color:transparent}.btn-block{width:100%;display:flex}.btn-default{background-color:var(--font-color);border-color:var(--invert-font-color);color:var(--invert-font-color)}.btn-default:focus:not(.btn-ghost),.btn-default:hover{background-color:var(--secondary-color);color:var(--invert-font-color)}.btn-default.btn-ghost:focus,.btn-default.btn-ghost:hover{border-color:var(--secondary-color);color:var(--secondary-color);z-index:2}.btn-error{color:var(--invert-font-color);background-color:var(--error-color);border:1px solid var(--error-color)}.btn-error:focus:not(.btn-ghost),.btn-error:hover{background-color:var(--error-color);border-color:var(--error-color)}.btn-error.btn-ghost{border-color:var(--error-color);color:var(--error-color)}.btn-error.btn-ghost:focus,.btn-error.btn-ghost:hover{border-color:var(--error-color);color:var(--error-color);z-index:2}.btn-primary{color:var(--invert-font-color);background-color:var(--primary-color);border:1px solid var(--primary-color)}.btn-primary:focus:not(.btn-ghost),.btn-primary:hover{background-color:var(--primary-color);border-color:var(--primary-color)}.btn-primary.btn-ghost{border-color:var(--primary-color);color:var(--primary-color)}.btn-primary.btn-ghost:focus,.btn-primary.btn-ghost:hover{border-color:var(--primary-color);color:var(--primary-color);z-index:2}.btn-small{padding:.5em 1.3em!important;font-size:.9em!important}.btn-group{overflow:auto}.btn-group .btn{float:left}.btn-group .btn-ghost:not(:first-child){margin-left:-1px}.terminal-card{border:1px solid var(--secondary-color)}.terminal-card>header{color:var(--invert-font-color);text-align:center;background-color:var(--secondary-color);padding:.5em 0}.terminal-card>div:first-of-type{padding:var(--global-space)}.terminal-timeline{position:relative;padding-left:70px}.terminal-timeline::before{content:\' \';background:var(--secondary-color);display:inline-block;position:absolute;left:35px;width:2px;height:100%;z-index:400}.terminal-timeline .terminal-card{margin-bottom:25px}.terminal-timeline .terminal-card::before{content:\' \';background:var(--invert-font-color);border:2px solid var(--secondary-color);display:inline-block;position:absolute;margin-top:25px;left:26px;width:15px;height:15px;z-index:400}.terminal-alert{color:var(--font-color);padding:1em;border:1px solid var(--font-color);margin-bottom:var(--global-space)}.terminal-alert-error{color:var(--error-color);border-color:var(--error-color)}.terminal-alert-primary{color:var(--primary-color);border-color:var(--primary-color)}@media screen and (max-width:960px){label{display:block;width:100%}pre::-webkit-scrollbar{height:3px}}@media screen and (max-width:480px){form{width:100%}}@media only screen and (min-width:30em){.terminal-nav{flex-direction:row;align-items:center}.terminal-menu ul{flex-direction:row;justify-items:flex-end;align-items:center;justify-content:flex-end;margin-top:calc(var(--global-space) * 2)}.terminal-menu li{margin:0;margin-right:2em}.terminal-menu li:last-child{margin-right:0}}.terminal-media:not(:last-child){margin-bottom:1.25rem}.terminal-media-left{padding-right:var(--global-space)}.terminal-media-left,.terminal-media-right{display:table-cell;vertical-align:top}.terminal-media-right{padding-left:var(--global-space)}.terminal-media-body{display:table-cell;vertical-align:top}.terminal-media-heading{font-size:1em;font-weight:700}.terminal-media-content{margin-top:.3rem}.terminal-placeholder{background-color:var(--secondary-color);text-align:center;color:var(--font-color);font-size:1rem;border:1px solid var(--secondary-color)}figure>img{padding:0}.terminal-avatarholder{width:calc(var(--global-space) * 5);height:calc(var(--global-space) * 5)}.terminal-avatarholder img{padding:0}figure{margin:0}figure>figcaption{color:var(--secondary-color);text-align:center}.hljs{display:block;overflow-x:auto;padding:.5em;background:var(--block-background-color);color:var(--font-color)}.hljs-comment,.hljs-quote{color:var(--secondary-color)}.hljs-variable{color:var(--font-color)}.hljs-built_in,.hljs-keyword,.hljs-name,.hljs-selector-tag,.hljs-tag{color:var(--primary-color)}.hljs-addition,.hljs-attribute,.hljs-literal,.hljs-section,.hljs-string,.hljs-template-tag,.hljs-template-variable,.hljs-title,.hljs-type{color:var(--secondary-color)}.hljs-string{color:var(--secondary-color)}.hljs-deletion,.hljs-meta,.hljs-selector-attr,.hljs-selector-pseudo{color:var(--primary-color)}.hljs-doctag{color:var(--secondary-color)}.hljs-attr{color:var(--primary-color)}.hljs-bullet,.hljs-link,.hljs-symbol{color:var(--primary-color)}.hljs-emphasis{font-style:italic}.hljs-strong{font-weight:700}',""]);const l=n},645:e=>{e.exports=function(e){var t=[];return t.toString=function(){return this.map((function(t){var o="",r=void 0!==t[5];return t[4]&&(o+="@supports (".concat(t[4],") {")),t[2]&&(o+="@media ".concat(t[2]," {")),r&&(o+="@layer".concat(t[5].length>0?" ".concat(t[5]):""," {")),o+=e(t),r&&(o+="}"),t[2]&&(o+="}"),t[4]&&(o+="}"),o})).join("")},t.i=function(e,o,r,i,a){"string"==typeof e&&(e=[[null,e,void 0]]);var n={};if(r)for(var l=0;l<this.length;l++){var s=this[l][0];null!=s&&(n[s]=!0)}for(var c=0;c<e.length;c++){var d=[].concat(e[c]);r&&n[d[0]]||(void 0!==a&&(void 0===d[5]||(d[1]="@layer".concat(d[5].length>0?" ".concat(d[5]):""," {").concat(d[1],"}")),d[5]=a),o&&(d[2]?(d[1]="@media ".concat(d[2]," {").concat(d[1],"}"),d[2]=o):d[2]=o),i&&(d[4]?(d[1]="@supports (".concat(d[4],") {").concat(d[1],"}"),d[4]=i):d[4]="".concat(i)),t.push(d))}},t}},81:e=>{e.exports=function(e){return e[1]}},379:e=>{var t=[];function o(e){for(var o=-1,r=0;r<t.length;r++)if(t[r].identifier===e){o=r;break}return o}function r(e,r){for(var a={},n=[],l=0;l<e.length;l++){var s=e[l],c=r.base?s[0]+r.base:s[0],d=a[c]||0,p="".concat(c," ").concat(d);a[c]=d+1;var h=o(p),u={css:s[1],media:s[2],sourceMap:s[3],supports:s[4],layer:s[5]};if(-1!==h)t[h].references++,t[h].updater(u);else{var v=i(u,r);r.byIndex=l,t.splice(l,0,{identifier:p,updater:v,references:1})}n.push(p)}return n}function i(e,t){var o=t.domAPI(t);return o.update(e),function(t){if(t){if(t.css===e.css&&t.media===e.media&&t.sourceMap===e.sourceMap&&t.supports===e.supports&&t.layer===e.layer)return;o.update(e=t)}else o.remove()}}e.exports=function(e,i){var a=r(e=e||[],i=i||{});return function(e){e=e||[];for(var n=0;n<a.length;n++){var l=o(a[n]);t[l].references--}for(var s=r(e,i),c=0;c<a.length;c++){var d=o(a[c]);0===t[d].references&&(t[d].updater(),t.splice(d,1))}a=s}}},569:e=>{var t={};e.exports=function(e,o){var r=function(e){if(void 0===t[e]){var o=document.querySelector(e);if(window.HTMLIFrameElement&&o instanceof window.HTMLIFrameElement)try{o=o.contentDocument.head}catch(e){o=null}t[e]=o}return t[e]}(e);if(!r)throw new Error("Couldn't find a style target. This probably means that the value for the 'insert' parameter is invalid.");r.appendChild(o)}},216:e=>{e.exports=function(e){var t=document.createElement("style");return e.setAttributes(t,e.attributes),e.insert(t,e.options),t}},565:(e,t,o)=>{e.exports=function(e){var t=o.nc;t&&e.setAttribute("nonce",t)}},795:e=>{e.exports=function(e){var t=e.insertStyleElement(e);return{update:function(o){!function(e,t,o){var r="";o.supports&&(r+="@supports (".concat(o.supports,") {")),o.media&&(r+="@media ".concat(o.media," {"));var i=void 0!==o.layer;i&&(r+="@layer".concat(o.layer.length>0?" ".concat(o.layer):""," {")),r+=o.css,i&&(r+="}"),o.media&&(r+="}"),o.supports&&(r+="}");var a=o.sourceMap;a&&"undefined"!=typeof btoa&&(r+="\n/*# sourceMappingURL=data:application/json;base64,".concat(btoa(unescape(encodeURIComponent(JSON.stringify(a))))," */")),t.styleTagTransform(r,e,t.options)}(t,e,o)},remove:function(){!function(e){if(null===e.parentNode)return!1;e.parentNode.removeChild(e)}(t)}}}},589:e=>{e.exports=function(e,t){if(t.styleSheet)t.styleSheet.cssText=e;else{for(;t.firstChild;)t.removeChild(t.firstChild);t.appendChild(document.createTextNode(e))}}}},t={};function o(r){var i=t[r];if(void 0!==i)return i.exports;var a=t[r]={id:r,exports:{}};return e[r](a,a.exports,o),a.exports}o.n=e=>{var t=e&&e.__esModule?()=>e.default:()=>e;return o.d(t,{a:t}),t},o.d=(e,t)=>{for(var r in t)o.o(t,r)&&!o.o(e,r)&&Object.defineProperty(e,r,{enumerable:!0,get:t[r]})},o.o=(e,t)=>Object.prototype.hasOwnProperty.call(e,t),o.nc=void 0,(()=>{var e=o(379),t=o.n(e),r=o(795),i=o.n(r),a=o(569),n=o.n(a),l=o(565),s=o.n(l),c=o(216),d=o.n(c),p=o(589),h=o.n(p),u=o(408),v={};v.styleTagTransform=h(),v.setAttributes=s(),v.insert=n().bind(null,"head"),v.domAPI=i(),v.insertStyleElement=d(),t()(u.Z,v),u.Z&&u.Z.locals&&u.Z.locals;const m=window.ShadowRoot&&(void 0===window.ShadyCSS||window.ShadyCSS.nativeShadow)&&"adoptedStyleSheets"in Document.prototype&&"replace"in CSSStyleSheet.prototype,f=Symbol(),g=new Map;class b{constructor(e,t){if(this._$cssResult$=!0,t!==f)throw Error("CSSResult is not constructable. Use `unsafeCSS` or `css` instead.");this.cssText=e}get styleSheet(){let e=g.get(this.cssText);return m&&void 0===e&&(g.set(this.cssText,e=new CSSStyleSheet),e.replaceSync(this.cssText)),e}toString(){return this.cssText}}const y=(e,...t)=>{const o=1===e.length?e[0]:t.reduce(((t,o,r)=>t+(e=>{if(!0===e._$cssResult$)return e.cssText;if("number"==typeof e)return e;throw Error("Value passed to 'css' function must be a 'css' function result: "+e+". Use 'unsafeCSS' to pass non-literal values, but take care to ensure page security.")})(o)+e[r+1]),e[0]);return new b(o,f)},w=m?e=>e:e=>e instanceof CSSStyleSheet?(e=>{let t="";for(const o of e.cssRules)t+=o.cssText;return(e=>new b("string"==typeof e?e:e+"",f))(t)})(e):e;var x;const A=window.trustedTypes,k=A?A.emptyScript:"",$=window.reactiveElementPolyfillSupport,_={toAttribute(e,t){switch(t){case Boolean:e=e?k:null;break;case Object:case Array:e=null==e?e:JSON.stringify(e)}return e},fromAttribute(e,t){let o=e;switch(t){case Boolean:o=null!==e;break;case Number:o=null===e?null:Number(e);break;case Object:case Array:try{o=JSON.parse(e)}catch(e){o=null}}return o}},S=(e,t)=>t!==e&&(t==t||e==e),E={attribute:!0,type:String,converter:_,reflect:!1,hasChanged:S};class C extends HTMLElement{constructor(){super(),this._$Et=new Map,this.isUpdatePending=!1,this.hasUpdated=!1,this._$Ei=null,this.o()}static addInitializer(e){var t;null!==(t=this.l)&&void 0!==t||(this.l=[]),this.l.push(e)}static get observedAttributes(){this.finalize();const e=[];return this.elementProperties.forEach(((t,o)=>{const r=this._$Eh(o,t);void 0!==r&&(this._$Eu.set(r,o),e.push(r))})),e}static createProperty(e,t=E){if(t.state&&(t.attribute=!1),this.finalize(),this.elementProperties.set(e,t),!t.noAccessor&&!this.prototype.hasOwnProperty(e)){const o="symbol"==typeof e?Symbol():"__"+e,r=this.getPropertyDescriptor(e,o,t);void 0!==r&&Object.defineProperty(this.prototype,e,r)}}static getPropertyDescriptor(e,t,o){return{get(){return this[t]},set(r){const i=this[e];this[t]=r,this.requestUpdate(e,i,o)},configurable:!0,enumerable:!0}}static getPropertyOptions(e){return this.elementProperties.get(e)||E}static finalize(){if(this.hasOwnProperty("finalized"))return!1;this.finalized=!0;const e=Object.getPrototypeOf(this);if(e.finalize(),this.elementProperties=new Map(e.elementProperties),this._$Eu=new Map,this.hasOwnProperty("properties")){const e=this.properties,t=[...Object.getOwnPropertyNames(e),...Object.getOwnPropertySymbols(e)];for(const o of t)this.createProperty(o,e[o])}return this.elementStyles=this.finalizeStyles(this.styles),!0}static finalizeStyles(e){const t=[];if(Array.isArray(e)){const o=new Set(e.flat(1/0).reverse());for(const e of o)t.unshift(w(e))}else void 0!==e&&t.push(w(e));return t}static _$Eh(e,t){const o=t.attribute;return!1===o?void 0:"string"==typeof o?o:"string"==typeof e?e.toLowerCase():void 0}o(){var e;this._$Ep=new Promise((e=>this.enableUpdating=e)),this._$AL=new Map,this._$Em(),this.requestUpdate(),null===(e=this.constructor.l)||void 0===e||e.forEach((e=>e(this)))}addController(e){var t,o;(null!==(t=this._$Eg)&&void 0!==t?t:this._$Eg=[]).push(e),void 0!==this.renderRoot&&this.isConnected&&(null===(o=e.hostConnected)||void 0===o||o.call(e))}removeController(e){var t;null===(t=this._$Eg)||void 0===t||t.splice(this._$Eg.indexOf(e)>>>0,1)}_$Em(){this.constructor.elementProperties.forEach(((e,t)=>{this.hasOwnProperty(t)&&(this._$Et.set(t,this[t]),delete this[t])}))}createRenderRoot(){var e;const t=null!==(e=this.shadowRoot)&&void 0!==e?e:this.attachShadow(this.constructor.shadowRootOptions);return((e,t)=>{m?e.adoptedStyleSheets=t.map((e=>e instanceof CSSStyleSheet?e:e.styleSheet)):t.forEach((t=>{const o=document.createElement("style"),r=window.litNonce;void 0!==r&&o.setAttribute("nonce",r),o.textContent=t.cssText,e.appendChild(o)}))})(t,this.constructor.elementStyles),t}connectedCallback(){var e;void 0===this.renderRoot&&(this.renderRoot=this.createRenderRoot()),this.enableUpdating(!0),null===(e=this._$Eg)||void 0===e||e.forEach((e=>{var t;return null===(t=e.hostConnected)||void 0===t?void 0:t.call(e)}))}enableUpdating(e){}disconnectedCallback(){var e;null===(e=this._$Eg)||void 0===e||e.forEach((e=>{var t;return null===(t=e.hostDisconnected)||void 0===t?void 0:t.call(e)}))}attributeChangedCallback(e,t,o){this._$AK(e,o)}_$ES(e,t,o=E){var r,i;const a=this.constructor._$Eh(e,o);if(void 0!==a&&!0===o.reflect){const n=(null!==(i=null===(r=o.converter)||void 0===r?void 0:r.toAttribute)&&void 0!==i?i:_.toAttribute)(t,o.type);this._$Ei=e,null==n?this.removeAttribute(a):this.setAttribute(a,n),this._$Ei=null}}_$AK(e,t){var o,r,i;const a=this.constructor,n=a._$Eu.get(e);if(void 0!==n&&this._$Ei!==n){const e=a.getPropertyOptions(n),l=e.converter,s=null!==(i=null!==(r=null===(o=l)||void 0===o?void 0:o.fromAttribute)&&void 0!==r?r:"function"==typeof l?l:null)&&void 0!==i?i:_.fromAttribute;this._$Ei=n,this[n]=s(t,e.type),this._$Ei=null}}requestUpdate(e,t,o){let r=!0;void 0!==e&&(((o=o||this.constructor.getPropertyOptions(e)).hasChanged||S)(this[e],t)?(this._$AL.has(e)||this._$AL.set(e,t),!0===o.reflect&&this._$Ei!==e&&(void 0===this._$EC&&(this._$EC=new Map),this._$EC.set(e,o))):r=!1),!this.isUpdatePending&&r&&(this._$Ep=this._$E_())}async _$E_(){this.isUpdatePending=!0;try{await this._$Ep}catch(e){Promise.reject(e)}const e=this.scheduleUpdate();return null!=e&&await e,!this.isUpdatePending}scheduleUpdate(){return this.performUpdate()}performUpdate(){var e;if(!this.isUpdatePending)return;this.hasUpdated,this._$Et&&(this._$Et.forEach(((e,t)=>this[t]=e)),this._$Et=void 0);let t=!1;const o=this._$AL;try{t=this.shouldUpdate(o),t?(this.willUpdate(o),null===(e=this._$Eg)||void 0===e||e.forEach((e=>{var t;return null===(t=e.hostUpdate)||void 0===t?void 0:t.call(e)})),this.update(o)):this._$EU()}catch(e){throw t=!1,this._$EU(),e}t&&this._$AE(o)}willUpdate(e){}_$AE(e){var t;null===(t=this._$Eg)||void 0===t||t.forEach((e=>{var t;return null===(t=e.hostUpdated)||void 0===t?void 0:t.call(e)})),this.hasUpdated||(this.hasUpdated=!0,this.firstUpdated(e)),this.updated(e)}_$EU(){this._$AL=new Map,this.isUpdatePending=!1}get updateComplete(){return this.getUpdateComplete()}getUpdateComplete(){return this._$Ep}shouldUpdate(e){return!0}update(e){void 0!==this._$EC&&(this._$EC.forEach(((e,t)=>this._$ES(t,this[t],e))),this._$EC=void 0),this._$EU()}updated(e){}firstUpdated(e){}}var z;C.finalized=!0,C.elementProperties=new Map,C.elementStyles=[],C.shadowRootOptions={mode:"open"},null==$||$({ReactiveElement:C}),(null!==(x=globalThis.reactiveElementVersions)&&void 0!==x?x:globalThis.reactiveElementVersions=[]).push("1.3.2");const j=globalThis.trustedTypes,P=j?j.createPolicy("lit-html",{createHTML:e=>e}):void 0,R=`lit$${(Math.random()+"").slice(9)}$`,N="?"+R,L=`<${N}>`,O=document,T=(e="")=>O.createComment(e),M=e=>null===e||"object"!=typeof e&&"function"!=typeof e,U=Array.isArray,H=/<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g,I=/-->/g,q=/>/g,D=/>|[ 	\n\r](?:([^\s"'>=/]+)([ 	\n\r]*=[ 	\n\r]*(?:[^ 	\n\r"'`<>=]|("|')|))|$)/g,V=/'/g,B=/"/g,F=/^(?:script|style|textarea|title)$/i,G=e=>(t,...o)=>({_$litType$:e,strings:t,values:o}),K=G(1),W=(G(2),Symbol.for("lit-noChange")),Z=Symbol.for("lit-nothing"),J=new WeakMap,X=O.createTreeWalker(O,129,null,!1),Q=(e,t)=>{const o=e.length-1,r=[];let i,a=2===t?"<svg>":"",n=H;for(let t=0;t<o;t++){const o=e[t];let l,s,c=-1,d=0;for(;d<o.length&&(n.lastIndex=d,s=n.exec(o),null!==s);)d=n.lastIndex,n===H?"!--"===s[1]?n=I:void 0!==s[1]?n=q:void 0!==s[2]?(F.test(s[2])&&(i=RegExp("</"+s[2],"g")),n=D):void 0!==s[3]&&(n=D):n===D?">"===s[0]?(n=null!=i?i:H,c=-1):void 0===s[1]?c=-2:(c=n.lastIndex-s[2].length,l=s[1],n=void 0===s[3]?D:'"'===s[3]?B:V):n===B||n===V?n=D:n===I||n===q?n=H:(n=D,i=void 0);const p=n===D&&e[t+1].startsWith("/>")?" ":"";a+=n===H?o+L:c>=0?(r.push(l),o.slice(0,c)+"$lit$"+o.slice(c)+R+p):o+R+(-2===c?(r.push(void 0),t):p)}const l=a+(e[o]||"<?>")+(2===t?"</svg>":"");if(!Array.isArray(e)||!e.hasOwnProperty("raw"))throw Error("invalid template strings array");return[void 0!==P?P.createHTML(l):l,r]};class Y{constructor({strings:e,_$litType$:t},o){let r;this.parts=[];let i=0,a=0;const n=e.length-1,l=this.parts,[s,c]=Q(e,t);if(this.el=Y.createElement(s,o),X.currentNode=this.el.content,2===t){const e=this.el.content,t=e.firstChild;t.remove(),e.append(...t.childNodes)}for(;null!==(r=X.nextNode())&&l.length<n;){if(1===r.nodeType){if(r.hasAttributes()){const e=[];for(const t of r.getAttributeNames())if(t.endsWith("$lit$")||t.startsWith(R)){const o=c[a++];if(e.push(t),void 0!==o){const e=r.getAttribute(o.toLowerCase()+"$lit$").split(R),t=/([.?@])?(.*)/.exec(o);l.push({type:1,index:i,name:t[2],strings:e,ctor:"."===t[1]?ie:"?"===t[1]?ne:"@"===t[1]?le:re})}else l.push({type:6,index:i})}for(const t of e)r.removeAttribute(t)}if(F.test(r.tagName)){const e=r.textContent.split(R),t=e.length-1;if(t>0){r.textContent=j?j.emptyScript:"";for(let o=0;o<t;o++)r.append(e[o],T()),X.nextNode(),l.push({type:2,index:++i});r.append(e[t],T())}}}else if(8===r.nodeType)if(r.data===N)l.push({type:2,index:i});else{let e=-1;for(;-1!==(e=r.data.indexOf(R,e+1));)l.push({type:7,index:i}),e+=R.length-1}i++}}static createElement(e,t){const o=O.createElement("template");return o.innerHTML=e,o}}function ee(e,t,o=e,r){var i,a,n,l;if(t===W)return t;let s=void 0!==r?null===(i=o._$Cl)||void 0===i?void 0:i[r]:o._$Cu;const c=M(t)?void 0:t._$litDirective$;return(null==s?void 0:s.constructor)!==c&&(null===(a=null==s?void 0:s._$AO)||void 0===a||a.call(s,!1),void 0===c?s=void 0:(s=new c(e),s._$AT(e,o,r)),void 0!==r?(null!==(n=(l=o)._$Cl)&&void 0!==n?n:l._$Cl=[])[r]=s:o._$Cu=s),void 0!==s&&(t=ee(e,s._$AS(e,t.values),s,r)),t}class te{constructor(e,t){this.v=[],this._$AN=void 0,this._$AD=e,this._$AM=t}get parentNode(){return this._$AM.parentNode}get _$AU(){return this._$AM._$AU}p(e){var t;const{el:{content:o},parts:r}=this._$AD,i=(null!==(t=null==e?void 0:e.creationScope)&&void 0!==t?t:O).importNode(o,!0);X.currentNode=i;let a=X.nextNode(),n=0,l=0,s=r[0];for(;void 0!==s;){if(n===s.index){let t;2===s.type?t=new oe(a,a.nextSibling,this,e):1===s.type?t=new s.ctor(a,s.name,s.strings,this,e):6===s.type&&(t=new se(a,this,e)),this.v.push(t),s=r[++l]}n!==(null==s?void 0:s.index)&&(a=X.nextNode(),n++)}return i}m(e){let t=0;for(const o of this.v)void 0!==o&&(void 0!==o.strings?(o._$AI(e,o,t),t+=o.strings.length-2):o._$AI(e[t])),t++}}class oe{constructor(e,t,o,r){var i;this.type=2,this._$AH=Z,this._$AN=void 0,this._$AA=e,this._$AB=t,this._$AM=o,this.options=r,this._$Cg=null===(i=null==r?void 0:r.isConnected)||void 0===i||i}get _$AU(){var e,t;return null!==(t=null===(e=this._$AM)||void 0===e?void 0:e._$AU)&&void 0!==t?t:this._$Cg}get parentNode(){let e=this._$AA.parentNode;const t=this._$AM;return void 0!==t&&11===e.nodeType&&(e=t.parentNode),e}get startNode(){return this._$AA}get endNode(){return this._$AB}_$AI(e,t=this){e=ee(this,e,t),M(e)?e===Z||null==e||""===e?(this._$AH!==Z&&this._$AR(),this._$AH=Z):e!==this._$AH&&e!==W&&this.$(e):void 0!==e._$litType$?this.T(e):void 0!==e.nodeType?this.k(e):(e=>{var t;return U(e)||"function"==typeof(null===(t=e)||void 0===t?void 0:t[Symbol.iterator])})(e)?this.S(e):this.$(e)}M(e,t=this._$AB){return this._$AA.parentNode.insertBefore(e,t)}k(e){this._$AH!==e&&(this._$AR(),this._$AH=this.M(e))}$(e){this._$AH!==Z&&M(this._$AH)?this._$AA.nextSibling.data=e:this.k(O.createTextNode(e)),this._$AH=e}T(e){var t;const{values:o,_$litType$:r}=e,i="number"==typeof r?this._$AC(e):(void 0===r.el&&(r.el=Y.createElement(r.h,this.options)),r);if((null===(t=this._$AH)||void 0===t?void 0:t._$AD)===i)this._$AH.m(o);else{const e=new te(i,this),t=e.p(this.options);e.m(o),this.k(t),this._$AH=e}}_$AC(e){let t=J.get(e.strings);return void 0===t&&J.set(e.strings,t=new Y(e)),t}S(e){U(this._$AH)||(this._$AH=[],this._$AR());const t=this._$AH;let o,r=0;for(const i of e)r===t.length?t.push(o=new oe(this.M(T()),this.M(T()),this,this.options)):o=t[r],o._$AI(i),r++;r<t.length&&(this._$AR(o&&o._$AB.nextSibling,r),t.length=r)}_$AR(e=this._$AA.nextSibling,t){var o;for(null===(o=this._$AP)||void 0===o||o.call(this,!1,!0,t);e&&e!==this._$AB;){const t=e.nextSibling;e.remove(),e=t}}setConnected(e){var t;void 0===this._$AM&&(this._$Cg=e,null===(t=this._$AP)||void 0===t||t.call(this,e))}}class re{constructor(e,t,o,r,i){this.type=1,this._$AH=Z,this._$AN=void 0,this.element=e,this.name=t,this._$AM=r,this.options=i,o.length>2||""!==o[0]||""!==o[1]?(this._$AH=Array(o.length-1).fill(new String),this.strings=o):this._$AH=Z}get tagName(){return this.element.tagName}get _$AU(){return this._$AM._$AU}_$AI(e,t=this,o,r){const i=this.strings;let a=!1;if(void 0===i)e=ee(this,e,t,0),a=!M(e)||e!==this._$AH&&e!==W,a&&(this._$AH=e);else{const r=e;let n,l;for(e=i[0],n=0;n<i.length-1;n++)l=ee(this,r[o+n],t,n),l===W&&(l=this._$AH[n]),a||(a=!M(l)||l!==this._$AH[n]),l===Z?e=Z:e!==Z&&(e+=(null!=l?l:"")+i[n+1]),this._$AH[n]=l}a&&!r&&this.C(e)}C(e){e===Z?this.element.removeAttribute(this.name):this.element.setAttribute(this.name,null!=e?e:"")}}class ie extends re{constructor(){super(...arguments),this.type=3}C(e){this.element[this.name]=e===Z?void 0:e}}const ae=j?j.emptyScript:"";class ne extends re{constructor(){super(...arguments),this.type=4}C(e){e&&e!==Z?this.element.setAttribute(this.name,ae):this.element.removeAttribute(this.name)}}class le extends re{constructor(e,t,o,r,i){super(e,t,o,r,i),this.type=5}_$AI(e,t=this){var o;if((e=null!==(o=ee(this,e,t,0))&&void 0!==o?o:Z)===W)return;const r=this._$AH,i=e===Z&&r!==Z||e.capture!==r.capture||e.once!==r.once||e.passive!==r.passive,a=e!==Z&&(r===Z||i);i&&this.element.removeEventListener(this.name,this,r),a&&this.element.addEventListener(this.name,this,e),this._$AH=e}handleEvent(e){var t,o;"function"==typeof this._$AH?this._$AH.call(null!==(o=null===(t=this.options)||void 0===t?void 0:t.host)&&void 0!==o?o:this.element,e):this._$AH.handleEvent(e)}}class se{constructor(e,t,o){this.element=e,this.type=6,this._$AN=void 0,this._$AM=t,this.options=o}get _$AU(){return this._$AM._$AU}_$AI(e){ee(this,e)}}const ce=window.litHtmlPolyfillSupport;var de,pe;null==ce||ce(Y,oe),(null!==(z=globalThis.litHtmlVersions)&&void 0!==z?z:globalThis.litHtmlVersions=[]).push("2.2.3");class he extends C{constructor(){super(...arguments),this.renderOptions={host:this},this._$Dt=void 0}createRenderRoot(){var e,t;const o=super.createRenderRoot();return null!==(e=(t=this.renderOptions).renderBefore)&&void 0!==e||(t.renderBefore=o.firstChild),o}update(e){const t=this.render();this.hasUpdated||(this.renderOptions.isConnected=this.isConnected),super.update(e),this._$Dt=((e,t,o)=>{var r,i;const a=null!==(r=null==o?void 0:o.renderBefore)&&void 0!==r?r:t;let n=a._$litPart$;if(void 0===n){const e=null!==(i=null==o?void 0:o.renderBefore)&&void 0!==i?i:null;a._$litPart$=n=new oe(t.insertBefore(T(),e),e,void 0,null!=o?o:{})}return n._$AI(e),n})(t,this.renderRoot,this.renderOptions)}connectedCallback(){var e;super.connectedCallback(),null===(e=this._$Dt)||void 0===e||e.setConnected(!0)}disconnectedCallback(){var e;super.disconnectedCallback(),null===(e=this._$Dt)||void 0===e||e.setConnected(!1)}render(){return W}}he.finalized=!0,he._$litElement$=!0,null===(de=globalThis.litElementHydrateSupport)||void 0===de||de.call(globalThis,{LitElement:he});const ue=globalThis.litElementPolyfillSupport;null==ue||ue({LitElement:he}),(null!==(pe=globalThis.litElementVersions)&&void 0!==pe?pe:globalThis.litElementVersions=[]).push("3.2.0");const ve=e=>t=>"function"==typeof t?((e,t)=>(window.customElements.define(e,t),t))(e,t):((e,t)=>{const{kind:o,elements:r}=t;return{kind:o,elements:r,finisher(t){window.customElements.define(e,t)}}})(e,t),me=(e,t)=>"method"===t.kind&&t.descriptor&&!("value"in t.descriptor)?{...t,finisher(o){o.createProperty(t.key,e)}}:{kind:"field",key:Symbol(),placement:"own",descriptor:{},originalKey:t.key,initializer(){"function"==typeof t.initializer&&(this[t.key]=t.initializer.call(this))},finisher(o){o.createProperty(t.key,e)}};function fe(e){return(t,o)=>void 0!==o?((e,t,o)=>{t.constructor.createProperty(o,e)})(e,t,o):me(e,t)}var ge;null===(ge=window.HTMLSlotElement)||void 0===ge||ge.prototype.assignedElements;class be extends he{get _slottedChildren(){const e=this.shadowRoot.querySelector("slot");if(e)return e.assignedElements({flatten:!0})}}const ye="categoryActivated",we=y`
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
`;var xe=function(e,t,o,r){var i,a=arguments.length,n=a<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,o,r);else for(var l=e.length-1;l>=0;l--)(i=e[l])&&(n=(a<3?i(n):a>3?i(t,o,n):i(t,o))||n);return a>3&&n&&Object.defineProperty(t,o,n),n};let Ae=class extends be{render(){return K`
      <ul @categoryActivated=${this._categoryActivatedListener}>
        <slot></slot>
      </ul>
    `}firstUpdated(){setTimeout((()=>{const e=new CustomEvent(ye,{bubbles:!0,composed:!0,detail:{id:this.default,description:"All the categories, for those who like a party."}});this.dispatchEvent(e),this._categoryActivatedListener(e)}))}_categoryActivatedListener(e){for(let t=0;t<this._slottedChildren.length;t++){const o=this._slottedChildren[t];o.name!=e.detail.id?o.disableCategory():o.active||o.enableCategory()}}};Ae.styles=we,xe([fe()],Ae.prototype,"default",void 0),Ae=xe([ve("rule-category-navigation")],Ae);const ke=y`
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
`;var $e=function(e,t,o,r){var i,a=arguments.length,n=a<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,o,r);else for(var l=e.length-1;l>=0;l--)(i=e[l])&&(n=(a<3?i(n):a>3?i(t,o,n):i(t,o))||n);return a>3&&n&&Object.defineProperty(t,o,n),n};let _e=class extends be{disableCategory(){this.active=!1,this.requestUpdate()}enableCategory(){this.active=!0,this.requestUpdate()}toggleCategory(e=!0){if(this.active=!this.active,e){const e={detail:{id:this.name,description:this.description},bubbles:!0,composed:!0};this.dispatchEvent(new CustomEvent(ye,e))}this.requestUpdate()}render(){return K`
      <li>
        <a
          href="#"
          class="${this.active?"active":""}"
          @click=${this.toggleCategory}
        >
          <slot></slot>
        </a>
      </li>
    `}};_e.styles=ke,$e([fe({type:String})],_e.prototype,"name",void 0),$e([fe({type:Boolean})],_e.prototype,"default",void 0),$e([fe({type:String})],_e.prototype,"description",void 0),_e=$e([ve("rule-category-link")],_e);let Se=class extends be{static get styles(){return[y`
      .html-report {
        height: 100%;
      }
    `]}render(){return K`
      <div
        class="html-report"
        @categoryActivated=${this._categoryActivatedListener}
        @violationSelected=${this._violationSelectedListener}
      >
        <slot name="navigation"></slot>
        <slot name="reports"></slot>
      </div>
    `}_categoryActivatedListener(e){const t=document.querySelectorAll("category-report"),o=document.querySelectorAll("category-rule"),r=document.querySelectorAll("category-rules"),i=document.querySelector("violation-drawer"),a=this.shadowRoot.querySelector("slot").assignedElements({flatten:!0})[0].querySelector("nav").querySelector("#category-description");a&&(a.innerHTML=e.detail.description),t.forEach((t=>{t.id==e.detail.id?t.style.display="block":t.style.display="none"})),o.forEach((e=>{e.otherRuleSelected()})),r.forEach((t=>{t.id==e.detail.id&&t.rules&&t.rules.length<=0&&(t.isEmpty=!0)})),i&&i.hide()}_violationSelectedListener(){document.querySelector("violation-drawer").show()}};Se=function(e,t,o,r){var i,a=arguments.length,n=a<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,o,r);else for(var l=e.length-1;l>=0;l--)(i=e[l])&&(n=(a<3?i(n):a>3?i(t,o,n):i(t,o))||n);return a>3&&n&&Object.defineProperty(t,o,n),n}([ve("html-report")],Se);var Ee=function(e,t,o,r){var i,a=arguments.length,n=a<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,o,r);else for(var l=e.length-1;l>=0;l--)(i=e[l])&&(n=(a<3?i(n):a>3?i(t,o,n):i(t,o))||n);return a>3&&n&&Object.defineProperty(t,o,n),n};let Ce=class extends be{get results(){return this.shadowRoot.querySelector("slot").assignedElements({flatten:!0})}render(){return K`<slot></slot>`}};Ee([fe()],Ce.prototype,"id",void 0),Ce=Ee([ve("category-report")],Ce);const ze=y`
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
`;var je=function(e,t,o,r){var i,a=arguments.length,n=a<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,o,r);else for(var l=e.length-1;l>=0;l--)(i=e[l])&&(n=(a<3?i(n):a>3?i(t,o,n):i(t,o))||n);return a>3&&n&&Object.defineProperty(t,o,n),n};let Pe=class extends be{connectedCallback(){super.connectedCallback(),this._violationId=Math.random().toString(20).substring(2)}get violationId(){return this._violationId}render(){return K` <nav
        aria-label="Violation Navigation"
        class="violation ${this.selected?"selected":""}"
        @click=${this._violationClicked}
      >
        <div class="line">${this.startLine}</div>
        <div class="message">${this.path}</div>
      </nav>
      <div class="code-render">
        <slot></slot>
      </div>`}_violationClicked(){let e;this._renderedCode?e=this._renderedCode:(e=this._slottedChildren[0],this._renderedCode=e);const t={detail:{message:this.message,id:this.ruleId,startLine:this.startLine,startCol:this.startCol,endLine:this.endLine,endCol:this.endCol,path:this.path,category:this.category,howToFix:this.howToFix,violationId:this.violationId,renderedCode:e},bubbles:!0,composed:!0};this.dispatchEvent(new CustomEvent("violationSelected",t))}};Pe.styles=ze,je([fe({type:String})],Pe.prototype,"message",void 0),je([fe({type:String})],Pe.prototype,"category",void 0),je([fe({type:String})],Pe.prototype,"ruleId",void 0),je([fe({type:Number})],Pe.prototype,"startLine",void 0),je([fe({type:Number})],Pe.prototype,"startCol",void 0),je([fe({type:Number})],Pe.prototype,"endLine",void 0),je([fe({type:Number})],Pe.prototype,"endCol",void 0),je([fe({type:String})],Pe.prototype,"path",void 0),je([fe({type:String})],Pe.prototype,"howToFix",void 0),je([fe({type:Boolean})],Pe.prototype,"selected",void 0),Pe=je([ve("category-rule-result")],Pe);const Re=y`
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
`,Ne=K`
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
`,Le=K`
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
`;var Oe=function(e,t,o,r){var i,a=arguments.length,n=a<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,o,r);else for(var l=e.length-1;l>=0;l--)(i=e[l])&&(n=(a<3?i(n):a>3?i(t,o,n):i(t,o))||n);return a>3&&n&&Object.defineProperty(t,o,n),n};let Te=class extends be{otherRuleSelected(){this.open=!1,this._violations.style.display="none",this._expandState=!1,this._slottedChildren.forEach((e=>{e.selected=!1})),this.requestUpdate()}render(){let e;this.truncated&&(e=K`
        <div class="truncated">
          <strong>${this.numResults-this.maxViolations}</strong> more
          violations not rendered, There are just too many!
        </div>
      `);const t=this._expandState?Le:Ne;return K`
      <nav
        aria-label="Rules and Violations"
        class="details ${this._expandState?"open":""}"
      >
        <div class="summary" @click=${this._ruleSelected}>
          <span class="expand-state">${t}</span>
          <span class="rule-icon">${this.ruleIcon}</span>
          <span class="rule-description">${this.description}</span>
          <span class="rule-violation-count">${this.numResults}</span>
        </div>
        <div class="violations" @violationSelected=${this._violationSelected}>
          <slot name="results"></slot>
          ${e}
        </div>
      </nav>
    `}_ruleSelected(){if(this.open)this._violations.style.display="none",this._expandState=!1;else{this._violations.style.display="block";const e=this.parentElement.parentElement.offsetHeight-60*this.totalRulesViolated;this._violations.style.maxHeight=e+"px",this._expandState=!0}this.open=!this.open,this.dispatchEvent(new CustomEvent("ruleSelected",{bubbles:!0,composed:!0,detail:{id:this.ruleId}})),this.requestUpdate()}_violationSelected(e){this._slottedChildren.forEach((t=>{t.selected=e.detail.violationId==t.violationId}))}};var Me;Te.styles=Re,Oe([fe()],Te.prototype,"totalRulesViolated",void 0),Oe([fe()],Te.prototype,"maxViolations",void 0),Oe([fe()],Te.prototype,"truncated",void 0),Oe([fe()],Te.prototype,"ruleId",void 0),Oe([fe()],Te.prototype,"description",void 0),Oe([fe()],Te.prototype,"numResults",void 0),Oe([fe()],Te.prototype,"ruleIcon",void 0),Oe([fe()],Te.prototype,"open",void 0),Oe([(Me=".violations",(({finisher:e,descriptor:t})=>(o,r)=>{var i;if(void 0===r){const r=null!==(i=o.originalKey)&&void 0!==i?i:o.key,a=null!=t?{kind:"method",placement:"prototype",key:r,descriptor:t(o.key)}:{...o,key:r};return null!=e&&(a.finisher=function(t){e(t,r)}),a}{const i=o.constructor;void 0!==t&&Object.defineProperty(o,r,t(r)),null==e||e(i,r)}})({descriptor:e=>{const t={get(){var e,t;return null!==(t=null===(e=this.renderRoot)||void 0===e?void 0:e.querySelector(Me))&&void 0!==t?t:null},enumerable:!0,configurable:!0};return t}}))],Te.prototype,"_violations",void 0),Te=Oe([ve("category-rule")],Te);const Ue=y`
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
`;var He=function(e,t,o,r){var i,a=arguments.length,n=a<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,o,r);else for(var l=e.length-1;l>=0;l--)(i=e[l])&&(n=(a<3?i(n):a>3?i(t,o,n):i(t,o))||n);return a>3&&n&&Object.defineProperty(t,o,n),n};let Ie=class extends be{render(){return this.isEmpty?K`
        <section class="no-violations">
          <p>All good in here, no rules broken!</p>
        </section>
      `:K`
        <section @ruleSelected=${this._ruleSelected}>
          <ul class="rule">
            <slot></slot>
          </ul>
        </section>
      `}get rules(){const e=this.shadowRoot.querySelector("slot");if(e)return e.assignedElements({flatten:!0})}_ruleSelected(e){this.rules.forEach((t=>{t.ruleId!=e.detail.id&&t.otherRuleSelected()}))}};Ie.styles=Ue,He([fe()],Ie.prototype,"id",void 0),He([fe()],Ie.prototype,"isEmpty",void 0),Ie=He([ve("category-rules")],Ie);const qe=y`
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
`;let De=class extends be{static get styles(){const e=y``;return[qe,e]}render(){return K`
      <slot
        @violationSelected=${this._violationSelectedListener}
        name="violation"
      ></slot>
      <slot name="details"></slot>
    `}_violationSelectedListener(e){const t=this.shadowRoot.querySelectorAll("slot")[1].assignedElements({flatten:!0})[0];t.ruleId=e.detail.id,t.message=e.detail.message,t.code=e.detail.renderedCode,t.howToFix=e.detail.howToFix,t.category=e.detail.category,t.path=e.detail.path}};De=function(e,t,o,r){var i,a=arguments.length,n=a<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,o,r);else for(var l=e.length-1;l>=0;l--)(i=e[l])&&(n=(a<3?i(n):a>3?i(t,o,n):i(t,o))||n);return a>3&&n&&Object.defineProperty(t,o,n),n}([ve("result-grid")],De);const Ve=[qe,y`
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
  `];var Be,Fe=function(e,t,o,r){var i,a=arguments.length,n=a<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,o,r);else for(var l=e.length-1;l>=0;l--)(i=e[l])&&(n=(a<3?i(n):a>3?i(t,o,n):i(t,o))||n);return a>3&&n&&Object.defineProperty(t,o,n),n};let Ge=Be=class extends be{static replaceTicks(e){const t=/(`[^`]*`)/g,o=e.split(t),r=new Array;return o.forEach((e=>{e.match(t)?r.push(K`
          <span class="backtick-element">${e.replace(/`/g,"")}</span>`):""!=e&&r.push(K`${e}`)})),r}render(){return this._visible?K`
        <h2>${Be.replaceTicks(this.message)}</h2>
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
      `:K`
        <section class="select-violation">
          <p>Please select a rule violation from a category.</p>
        </section>
      `}get drawer(){return document.querySelector("violation-drawer")}show(){this._visible=!0,this.drawer.classList.add("drawer-active"),this.requestUpdate()}hide(){this._visible=!1,this.drawer.classList.remove("drawer-active"),this.requestUpdate()}};Ge.styles=Ve,Fe([fe({type:Element})],Ge.prototype,"code",void 0),Fe([fe({type:String})],Ge.prototype,"message",void 0),Fe([fe({type:String})],Ge.prototype,"path",void 0),Fe([fe({type:String})],Ge.prototype,"category",void 0),Fe([fe({type:String})],Ge.prototype,"ruleId",void 0),Fe([fe({type:String})],Ge.prototype,"howToFix",void 0),Ge=Be=Fe([ve("violation-drawer")],Ge);var Ke=function(e,t,o,r){var i,a=arguments.length,n=a<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,o):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,o,r);else for(var l=e.length-1;l>=0;l--)(i=e[l])&&(n=(a<3?i(n):a>3?i(t,o,n):i(t,o))||n);return a>3&&n&&Object.defineProperty(t,o,n),n};let We=class extends be{static get styles(){return[y`
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
    `]}render(){return K`
      <div class=${this.colorForScore()}>
        <span class="grade">${this.value}${this.percentage?"%":""}</span>
        <span class="label"> ${this.label} </span>
      </div>
    `}colorForScore(){if(this.preset)return this.preset;switch(!0){case this.value<=10:return"error";case this.value>10&&this.value<20:return"warn-400";case this.value>=20&&this.value<30:return"warn-300";case this.value>=30&&this.value<40:return"warn-200";case this.value>=40&&this.value<50:return"warn";case this.value>=50&&this.value<65:return"ok-400";case this.value>=65&&this.value<75:return"ok-300";case this.value>=75&&this.value<95:return"ok-200";case this.value>=95:default:return"ok"}}};Ke([fe()],We.prototype,"value",void 0),Ke([fe()],We.prototype,"preset",void 0),Ke([fe()],We.prototype,"percentage",void 0),Ke([fe()],We.prototype,"label",void 0),We=Ke([ve("header-statistic")],We)})()})();
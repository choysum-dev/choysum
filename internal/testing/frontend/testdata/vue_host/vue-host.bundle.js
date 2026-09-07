(() => {
  // choysum-esm:https://esm.sh/@vue/shared@3.5.42/es2020/shared.mjs?target=es2020
  function l(e) {
    let t2 = /* @__PURE__ */ Object.create(null);
    for (let n2 of e.split(",")) t2[n2] = 1;
    return (n2) => n2 in t2;
  }
  var se = {};
  var ae = [];
  var ce = () => {
  };
  var le = () => false;
  var pe = (e) => e.charCodeAt(0) === 111 && e.charCodeAt(1) === 110 && (e.charCodeAt(2) > 122 || e.charCodeAt(2) < 97);
  var fe = (e) => e.startsWith("onUpdate:");
  var de = Object.assign;
  var me = (e, t2) => {
    let n2 = e.indexOf(t2);
    n2 > -1 && e.splice(n2, 1);
  };
  var U = Object.prototype.hasOwnProperty;
  var ue = (e, t2) => U.call(e, t2);
  var d = Array.isArray;
  var S = (e) => m(e) === "[object Map]";
  var N = (e) => m(e) === "[object Set]";
  var k = (e) => m(e) === "[object Date]";
  var g = (e) => typeof e == "function";
  var p = (e) => typeof e == "string";
  var y = (e) => typeof e == "symbol";
  var f = (e) => e !== null && typeof e == "object";
  var ge = (e) => (f(e) || g(e)) && g(e.then) && g(e.catch);
  var L = Object.prototype.toString;
  var m = (e) => L.call(e);
  var ye = (e) => m(e).slice(8, -1);
  var z = (e) => m(e) === "[object Object]";
  var Ee = (e) => p(e) && e !== "NaN" && e[0] !== "-" && "" + parseInt(e, 10) === e;
  var be = l(",key,ref,ref_for,ref_key,onVnodeBeforeMount,onVnodeMounted,onVnodeBeforeUpdate,onVnodeUpdated,onVnodeBeforeUnmount,onVnodeUnmounted");
  var Te = l("bind,cloak,else-if,else,for,html,if,model,on,once,pre,show,slot,text,memo");
  var E = (e) => {
    let t2 = /* @__PURE__ */ Object.create(null);
    return ((n2) => t2[n2] || (t2[n2] = e(n2)));
  };
  var j = /-\w/g;
  var Ae = E((e) => e.replace(j, (t2) => t2.slice(1).toUpperCase()));
  var V = /\B([A-Z])/g;
  var H = E((e) => e.replace(V, "-$1").toLowerCase());
  var B = E((e) => e.charAt(0).toUpperCase() + e.slice(1));
  var Se = E((e) => e ? `on${B(e)}` : "");
  var Ne = (e, t2) => !Object.is(e, t2);
  var Oe = (e, ...t2) => {
    for (let n2 = 0; n2 < e.length; n2++) e[n2](...t2);
  };
  var xe = (e, t2, n2, r2 = false) => {
    Object.defineProperty(e, t2, { configurable: true, enumerable: false, writable: r2, value: n2 });
  };
  var ke = (e) => {
    let t2 = parseFloat(e);
    return isNaN(t2) ? e : t2;
  };
  var Ce = (e) => {
    let t2 = p(e) ? Number(e) : NaN;
    return isNaN(t2) ? e : t2;
  };
  var C;
  var _e = () => C || (C = typeof globalThis < "u" ? globalThis : typeof self < "u" ? self : typeof window < "u" ? window : typeof globalThis < "u" ? globalThis : {});
  var Y = "Infinity,undefined,NaN,isFinite,isNaN,parseFloat,parseInt,decodeURI,decodeURIComponent,encodeURI,encodeURIComponent,Math,Number,Date,Array,Object,Boolean,String,RegExp,Map,Set,JSON,Intl,BigInt,console,Error,Symbol";
  var v = l(Y);
  function M(e) {
    if (d(e)) {
      let t2 = {};
      for (let n2 = 0; n2 < e.length; n2++) {
        let r2 = e[n2], i = p(r2) ? W(r2) : M(r2);
        if (i) for (let o in i) t2[o] = i[o];
      }
      return t2;
    } else if (p(e) || f(e)) return e;
  }
  var K = /;(?![^(]*\))/g;
  var q = /:([^]+)/;
  var $ = /\/\*[^]*?\*\//g;
  function W(e) {
    let t2 = {};
    return e.replace($, "").split(K).forEach((n2) => {
      if (n2) {
        let r2 = n2.split(q);
        r2.length > 1 && (t2[r2[0].trim()] = r2[1].trim());
      }
    }), t2;
  }
  function w(e) {
    let t2 = "";
    if (p(e)) t2 = e;
    else if (d(e)) for (let n2 = 0; n2 < e.length; n2++) {
      let r2 = w(e[n2]);
      r2 && (t2 += r2 + " ");
    }
    else if (f(e)) for (let n2 in e) e[n2] && (t2 += n2 + " ");
    return t2.trim();
  }
  var X = "html,body,base,head,link,meta,style,title,address,article,aside,footer,header,hgroup,h1,h2,h3,h4,h5,h6,nav,section,div,dd,dl,dt,figcaption,figure,picture,hr,img,li,main,ol,p,pre,ul,a,b,abbr,bdi,bdo,br,cite,code,data,dfn,em,i,kbd,mark,q,rp,rt,ruby,s,samp,small,span,strong,sub,sup,time,u,var,wbr,area,audio,map,track,video,embed,object,param,source,canvas,script,noscript,del,ins,caption,col,colgroup,table,thead,tbody,td,th,tr,button,datalist,fieldset,form,input,label,legend,meter,optgroup,option,output,progress,select,textarea,details,dialog,menu,summary,template,blockquote,iframe,tfoot";
  var J = "svg,animate,animateMotion,animateTransform,circle,clipPath,color-profile,defs,desc,discard,ellipse,feBlend,feColorMatrix,feComponentTransfer,feComposite,feConvolveMatrix,feDiffuseLighting,feDisplacementMap,feDistantLight,feDropShadow,feFlood,feFuncA,feFuncB,feFuncG,feFuncR,feGaussianBlur,feImage,feMerge,feMergeNode,feMorphology,feOffset,fePointLight,feSpecularLighting,feSpotLight,feTile,feTurbulence,filter,foreignObject,g,hatch,hatchpath,image,line,linearGradient,marker,mask,mesh,meshgradient,meshpatch,meshrow,metadata,mpath,path,pattern,polygon,polyline,radialGradient,rect,set,solidcolor,stop,switch,symbol,text,textPath,title,tspan,unknown,use,view";
  var Z = "annotation,annotation-xml,maction,maligngroup,malignmark,math,menclose,merror,mfenced,mfrac,mfraction,mglyph,mi,mlabeledtr,mlongdiv,mmultiscripts,mn,mo,mover,mpadded,mphantom,mprescripts,mroot,mrow,ms,mscarries,mscarry,msgroup,msline,mspace,msqrt,msrow,mstack,mstyle,msub,msubsup,msup,mtable,mtd,mtext,mtr,munder,munderover,none,semantics";
  var Q = "area,base,br,col,embed,hr,img,input,link,meta,param,source,track,wbr";
  var Ve = l(X);
  var He = l(J);
  var Be = l(Z);
  var Ge = l(Q);
  var D = "itemscope,allowfullscreen,formnovalidate,ismap,nomodule,novalidate,readonly";
  var Ye = l(D);
  var ve = l(D + ",async,autofocus,autoplay,controls,default,defer,disabled,inert,loop,open,required,reversed,scoped,seamless,checked,muted,multiple,selected");
  function Ke(e) {
    return !!e || e === "";
  }
  var We = l("accept,accept-charset,accesskey,action,align,allow,alt,async,autocapitalize,autocomplete,autofocus,autoplay,background,bgcolor,border,buffered,capture,challenge,charset,checked,cite,class,code,codebase,color,cols,colspan,content,contenteditable,contextmenu,controls,coords,crossorigin,csp,data,datetime,decoding,default,defer,dir,dirname,disabled,download,draggable,dropzone,enctype,enterkeyhint,for,form,formaction,formenctype,formmethod,formnovalidate,formtarget,headers,height,hidden,high,href,hreflang,http-equiv,icon,id,importance,inert,integrity,ismap,itemprop,keytype,kind,label,lang,language,loading,list,loop,low,manifest,max,maxlength,minlength,media,min,multiple,muted,name,novalidate,open,optimum,pattern,ping,placeholder,poster,preload,radiogroup,readonly,referrerpolicy,rel,required,reversed,rows,rowspan,sandbox,scope,scoped,selected,shape,size,sizes,slot,span,spellcheck,src,srcdoc,srclang,srcset,start,step,style,summary,tabindex,target,title,translate,type,usemap,value,width,wrap");
  var Xe = l("xmlns,accent-height,accumulate,additive,alignment-baseline,alphabetic,amplitude,arabic-form,ascent,attributeName,attributeType,azimuth,baseFrequency,baseline-shift,baseProfile,bbox,begin,bias,by,calcMode,cap-height,class,clip,clipPathUnits,clip-path,clip-rule,color,color-interpolation,color-interpolation-filters,color-profile,color-rendering,contentScriptType,contentStyleType,crossorigin,cursor,cx,cy,d,decelerate,descent,diffuseConstant,direction,display,divisor,dominant-baseline,dur,dx,dy,edgeMode,elevation,enable-background,end,exponent,fill,fill-opacity,fill-rule,filter,filterRes,filterUnits,flood-color,flood-opacity,font-family,font-size,font-size-adjust,font-stretch,font-style,font-variant,font-weight,format,from,fr,fx,fy,g1,g2,glyph-name,glyph-orientation-horizontal,glyph-orientation-vertical,glyphRef,gradientTransform,gradientUnits,hanging,height,href,hreflang,horiz-adv-x,horiz-origin-x,id,ideographic,image-rendering,in,in2,intercept,k,k1,k2,k3,k4,kernelMatrix,kernelUnitLength,kerning,keyPoints,keySplines,keyTimes,lang,lengthAdjust,letter-spacing,lighting-color,limitingConeAngle,local,marker-end,marker-mid,marker-start,markerHeight,markerUnits,markerWidth,mask,maskContentUnits,maskUnits,mathematical,max,media,method,min,mode,name,numOctaves,offset,opacity,operator,order,orient,orientation,origin,overflow,overline-position,overline-thickness,panose-1,paint-order,path,pathLength,patternContentUnits,patternTransform,patternUnits,ping,pointer-events,points,pointsAtX,pointsAtY,pointsAtZ,preserveAlpha,preserveAspectRatio,primitiveUnits,r,radius,referrerPolicy,refX,refY,rel,rendering-intent,repeatCount,repeatDur,requiredExtensions,requiredFeatures,restart,result,rotate,rx,ry,scale,seed,shape-rendering,slope,spacing,specularConstant,specularExponent,speed,spreadMethod,startOffset,stdDeviation,stemh,stemv,stitchTiles,stop-color,stop-opacity,strikethrough-position,strikethrough-thickness,string,stroke,stroke-dasharray,stroke-dashoffset,stroke-linecap,stroke-linejoin,stroke-miterlimit,stroke-opacity,stroke-width,style,surfaceScale,systemLanguage,tabindex,tableValues,target,targetX,targetY,text-anchor,text-decoration,text-rendering,textLength,to,transform,transform-origin,type,u1,u2,underline-position,underline-thickness,unicode,unicode-bidi,unicode-range,units-per-em,v-alphabetic,v-hanging,v-ideographic,v-mathematical,values,vector-effect,version,vert-adv-y,vert-origin-x,vert-origin-y,viewBox,viewTarget,visibility,width,widths,word-spacing,writing-mode,x,x-height,x1,x2,xChannelSelector,xlink:actuate,xlink:arcrole,xlink:href,xlink:role,xlink:show,xlink:title,xlink:type,xmlns:xlink,xml:base,xml:lang,xml:space,y,y1,y2,yChannelSelector,z,zoomAndPan");
  var Je = l("accent,accentunder,actiontype,align,alignmentscope,altimg,altimg-height,altimg-valign,altimg-width,alttext,bevelled,close,columnsalign,columnlines,columnspan,denomalign,depth,dir,display,displaystyle,encoding,equalcolumns,equalrows,fence,fontstyle,fontweight,form,frame,framespacing,groupalign,height,href,id,indentalign,indentalignfirst,indentalignlast,indentshift,indentshiftfirst,indentshiftlast,indextype,justify,largetop,largeop,lquote,lspace,mathbackground,mathcolor,mathsize,mathvariant,maxsize,minlabelspacing,mode,other,overflow,position,rowalign,rowlines,rowspan,rquote,rspace,scriptlevel,scriptminsize,scriptsizemultiplier,selection,separator,separators,shift,side,src,stackalign,stretchy,subscriptshift,superscriptshift,symmetric,voffset,width,widths,xlink:href,xlink:show,xlink:type,xmlns");
  function oe(e, t2) {
    if (e.length !== t2.length) return false;
    let n2 = true;
    for (let r2 = 0; n2 && r2 < e.length; r2++) n2 = b(e[r2], t2[r2]);
    return n2;
  }
  function R(e, t2) {
    if (e.size !== t2.size) return false;
    let n2 = Array.from(t2), r2 = new Uint8Array(n2.length);
    for (let i of e) {
      let o = -1;
      for (let s = 0; s < n2.length; s++) if (!r2[s] && b(i, n2[s])) {
        o = s;
        break;
      }
      if (o < 0) return false;
      r2[o] = 1;
    }
    return true;
  }
  function b(e, t2) {
    if (e === t2) return true;
    let n2 = k(e), r2 = k(t2);
    if (n2 || r2) return n2 && r2 ? e.getTime() === t2.getTime() : false;
    if (n2 = y(e), r2 = y(t2), n2 || r2) return e === t2;
    if (n2 = d(e), r2 = d(t2), n2 || r2) return n2 && r2 ? oe(e, t2) : false;
    if (n2 = f(e), r2 = f(t2), n2 || r2) {
      if (!n2 || !r2) return false;
      if (n2 = S(e), r2 = S(t2), n2 || r2 || (n2 = N(e), r2 = N(t2), n2 || r2)) return n2 && r2 ? R(e, t2) : false;
      let i = Object.keys(e).length, o = Object.keys(t2).length;
      if (i !== o) return false;
      for (let s in e) {
        let c = e.hasOwnProperty(s), a = t2.hasOwnProperty(s);
        if (c && !a || !c && a || !b(e[s], t2[s])) return false;
      }
    }
    return String(e) === String(t2);
  }
  var P = (e) => !!(e && e.__v_isRef === true);
  var ie = (e) => p(e) ? e : e == null ? "" : d(e) || f(e) && (e.toString === L || !g(e.toString)) ? P(e) ? ie(e.value) : JSON.stringify(e, F, 2) : String(e);
  var F = (e, t2) => P(t2) ? F(e, t2.value) : S(t2) ? { [`Map(${t2.size})`]: [...t2.entries()].reduce((n2, [r2, i], o) => (n2[A(r2, o) + " =>"] = i, n2), {}) } : N(t2) ? { [`Set(${t2.size})`]: [...t2.values()].map((n2) => A(n2)) } : y(t2) ? A(t2) : f(t2) && !d(t2) && !z(t2) ? String(t2) : t2;
  var A = (e, t2 = "") => {
    var n2;
    return y(e) ? `Symbol(${(n2 = e.description) != null ? n2 : t2})` : e;
  };

  // choysum-esm:https://esm.sh/@vue/reactivity@3.5.42/es2020/reactivity.mjs?target=es2020
  function ct(e, ...t2) {
    console.warn(`[Vue warn] ${e}`, ...t2);
  }
  var v2;
  var ge2 = class {
    constructor(t2 = false) {
      this.detached = t2, this._active = true, this._on = 0, this.effects = [], this.cleanups = [], this._isPaused = false, this._warnOnRun = true, this.__v_skip = true, !t2 && v2 && (v2.active ? (this.parent = v2, this.index = (v2.scopes || (v2.scopes = [])).push(this) - 1) : (this._active = false, this._warnOnRun = false));
    }
    get active() {
      return this._active;
    }
    pause() {
      if (this._active) {
        this._isPaused = true;
        let t2, s;
        if (this.scopes) {
          let n2 = this.scopes.slice();
          for (t2 = 0, s = n2.length; t2 < s; t2++) n2[t2].pause();
        }
        for (t2 = 0, s = this.effects.length; t2 < s; t2++) this.effects[t2].pause();
      }
    }
    resume() {
      if (this._active && this._isPaused) {
        this._isPaused = false;
        let t2, s;
        if (this.scopes) {
          let i = this.scopes.slice();
          for (t2 = 0, s = i.length; t2 < s; t2++) i[t2].resume();
        }
        let n2 = this.effects.slice();
        for (t2 = 0, s = n2.length; t2 < s; t2++) n2[t2].resume();
      }
    }
    run(t2) {
      if (this._active) {
        let s = v2;
        try {
          return v2 = this, t2();
        } finally {
          v2 = s;
        }
      }
    }
    on() {
      ++this._on === 1 && (this.prevScope = v2, v2 = this);
    }
    off() {
      if (this._on > 0 && --this._on === 0) {
        if (v2 === this) v2 = this.prevScope;
        else {
          let t2 = v2;
          for (; t2; ) {
            if (t2.prevScope === this) {
              t2.prevScope = this.prevScope;
              break;
            }
            t2 = t2.prevScope;
          }
        }
        this.prevScope = void 0;
      }
    }
    stop(t2) {
      if (this._active) {
        this._active = false;
        let s, n2;
        for (s = 0, n2 = this.effects.length; s < n2; s++) this.effects[s].stop();
        for (this.effects.length = 0, s = 0, n2 = this.cleanups.length; s < n2; s++) this.cleanups[s]();
        if (this.cleanups.length = 0, this.scopes) {
          let i = this.scopes.slice();
          for (s = 0, n2 = i.length; s < n2; s++) i[s].stop(true);
          this.scopes.length = 0;
        }
        if (!this.detached && this.parent && !t2) {
          let i = this.parent.scopes.pop();
          i && i !== this && (this.parent.scopes[this.index] = i, i.index = this.index);
        }
        this.parent = void 0;
      }
    }
  };
  function ft() {
    return v2;
  }
  var h;
  var de2 = /* @__PURE__ */ new WeakSet();
  var k2 = class {
    constructor(t2) {
      this.fn = t2, this.deps = void 0, this.depsTail = void 0, this.flags = 5, this.next = void 0, this.cleanup = void 0, this.scheduler = void 0, v2 && (v2.active ? v2.effects.push(this) : this.flags &= -2);
    }
    pause() {
      this.flags |= 64;
    }
    resume() {
      this.flags & 64 && (this.flags &= -65, de2.has(this) && (de2.delete(this), this.trigger()));
    }
    notify() {
      this.flags & 2 && !(this.flags & 32) || this.flags & 8 || ke2(this);
    }
    run() {
      if (!(this.flags & 1)) return this.fn();
      this.flags |= 2, Pe(this), Ke2(this);
      let t2 = h, s = E2;
      h = this, E2 = true;
      try {
        return this.fn();
      } finally {
        He2(this), h = t2, E2 = s, this.flags &= -3;
      }
    }
    stop() {
      if (this.flags & 1) {
        for (let t2 = this.deps; t2; t2 = t2.nextDep) Ie(t2);
        this.deps = this.depsTail = void 0, Pe(this), this.onStop && this.onStop(), this.flags &= -2;
      }
    }
    trigger() {
      this.flags & 64 ? de2.add(this) : this.scheduler ? this.scheduler() : this.runIfDirty();
    }
    runIfDirty() {
      we(this) && this.run();
    }
    get dirty() {
      return we(this);
    }
  };
  var We2 = 0;
  var Y2;
  var F2;
  function ke2(e, t2 = false) {
    if (e.flags |= 8, t2) {
      e.next = F2, F2 = e;
      return;
    }
    e.next = Y2, Y2 = e;
  }
  function Ae2() {
    We2++;
  }
  function Ve2() {
    if (--We2 > 0) return;
    if (F2) {
      let t2 = F2;
      for (F2 = void 0; t2; ) {
        let s = t2.next;
        t2.next = void 0, t2.flags &= -9, t2 = s;
      }
    }
    let e;
    for (; Y2; ) {
      let t2 = Y2;
      for (Y2 = void 0; t2; ) {
        let s = t2.next;
        if (t2.next = void 0, t2.flags &= -9, t2.flags & 1) try {
          t2.trigger();
        } catch (n2) {
          e || (e = n2);
        }
        t2 = s;
      }
    }
    if (e) throw e;
  }
  function Ke2(e) {
    for (let t2 = e.deps; t2; t2 = t2.nextDep) t2.version = -1, t2.prevActiveLink = t2.dep.activeLink, t2.dep.activeLink = t2;
  }
  function He2(e) {
    let t2, s = e.depsTail, n2 = s;
    for (; n2; ) {
      let i = n2.prevDep;
      n2.version === -1 ? (n2 === s && (s = i), Ie(n2), lt(n2)) : t2 = n2, n2.dep.activeLink = n2.prevActiveLink, n2.prevActiveLink = void 0, n2 = i;
    }
    e.deps = t2, e.depsTail = s;
  }
  function we(e) {
    for (let t2 = e.deps; t2; t2 = t2.nextDep) if (t2.dep.version !== t2.version || t2.dep.computed && (je(t2.dep.computed) || t2.dep.version !== t2.version)) return true;
    return !!e._dirty;
  }
  function je(e) {
    if (e.flags & 4 && !(e.flags & 16) || (e.flags &= -17, e.globalVersion === B2) || (e.globalVersion = B2, !e.isSSR && e.flags & 128 && (!e.deps && !e._dirty || !we(e)))) return;
    e.flags |= 2;
    let t2 = e.dep, s = h, n2 = E2;
    h = e, E2 = true;
    try {
      Ke2(e);
      let i = e.fn(e._value);
      (t2.version === 0 || Ne(i, e._value)) && (e.flags |= 128, e._value = i, t2.version++);
    } catch (i) {
      throw t2.version++, i;
    } finally {
      h = s, E2 = n2, He2(e), e.flags &= -3;
    }
  }
  function Ie(e, t2 = false) {
    let { dep: s, prevSub: n2, nextSub: i } = e;
    if (n2 && (n2.nextSub = i, e.prevSub = void 0), i && (i.prevSub = n2, e.nextSub = void 0), s.subs === e && (s.subs = n2, !n2 && s.computed)) {
      s.computed.flags &= -5;
      for (let r2 = s.computed.deps; r2; r2 = r2.nextDep) Ie(r2, true);
    }
    !t2 && !--s.sc && s.map && s.map.delete(s.key);
  }
  function lt(e) {
    let { prevDep: t2, nextDep: s } = e;
    t2 && (t2.nextDep = s, e.prevDep = void 0), s && (s.prevDep = t2, e.nextDep = void 0);
  }
  var E2 = true;
  var me2 = [];
  function Ue() {
    me2.push(E2), E2 = false;
  }
  function $e() {
    let e = me2.pop();
    E2 = e === void 0 ? true : e;
  }
  function Pe(e) {
    let { cleanup: t2 } = e;
    if (e.cleanup = void 0, t2) {
      let s = h;
      h = void 0;
      try {
        t2();
      } finally {
        h = s;
      }
    }
  }
  var B2 = 0;
  var Ee2 = class {
    constructor(t2, s) {
      this.sub = t2, this.dep = s, this.version = s.version, this.nextDep = this.prevDep = this.nextSub = this.prevSub = this.prevActiveLink = void 0;
    }
  };
  var K2 = class {
    constructor(t2) {
      this.computed = t2, this.version = 0, this.activeLink = void 0, this.subs = void 0, this.map = void 0, this.key = void 0, this.sc = 0, this.__v_skip = true;
    }
    track(t2) {
      if (!h || !E2 || h === this.computed) return;
      let s = this.activeLink;
      if (s === void 0 || s.sub !== h) s = this.activeLink = new Ee2(h, this), h.deps ? (s.prevDep = h.depsTail, h.depsTail.nextDep = s, h.depsTail = s) : h.deps = h.depsTail = s, Ge2(s);
      else if (s.version === -1 && (s.version = this.version, s.nextDep)) {
        let n2 = s.nextDep;
        n2.prevDep = s.prevDep, s.prevDep && (s.prevDep.nextDep = n2), s.prevDep = h.depsTail, s.nextDep = void 0, h.depsTail.nextDep = s, h.depsTail = s, h.deps === s && (h.deps = n2);
      }
      return s;
    }
    trigger(t2) {
      this.version++, B2++, this.notify(t2);
    }
    notify(t2) {
      Ae2();
      try {
        for (let s = this.subs; s; s = s.prevSub) s.sub.notify() && s.sub.dep.notify();
      } finally {
        Ve2();
      }
    }
  };
  function Ge2(e) {
    if (e.dep.sc++, e.sub.flags & 4) {
      let t2 = e.dep.computed;
      if (t2 && !e.dep.subs) {
        t2.flags |= 20;
        for (let n2 = t2.deps; n2; n2 = n2.nextDep) Ge2(n2);
      }
      let s = e.dep.subs;
      s !== e && (e.prevSub = s, s && (s.nextSub = e)), e.dep.subs = e;
    }
  }
  var re = /* @__PURE__ */ new WeakMap();
  var m2 = /* @__PURE__ */ Symbol("");
  var ye2 = /* @__PURE__ */ Symbol("");
  var J2 = /* @__PURE__ */ Symbol("");
  function w2(e, t2, s) {
    if (E2 && h) {
      let n2 = re.get(e);
      n2 || re.set(e, n2 = /* @__PURE__ */ new Map());
      let i = n2.get(s);
      i || (n2.set(s, i = new K2()), i.map = n2, i.key = s), i.track();
    }
  }
  function x(e, t2, s, n2, i, r2) {
    let o = re.get(e);
    if (!o) {
      B2++;
      return;
    }
    let a = (c) => {
      c && c.trigger();
    };
    if (Ae2(), t2 === "clear") o.forEach(a);
    else {
      let c = d(e), u = c && Ee(s);
      if (c && s === "length") {
        let d2 = Number(n2);
        o.forEach((f2, _) => {
          (_ === "length" || _ === J2 || !y(_) && _ >= d2) && a(f2);
        });
      } else switch ((s !== void 0 || o.has(void 0)) && a(o.get(s)), u && a(o.get(J2)), t2) {
        case "add":
          c ? u && a(o.get("length")) : (a(o.get(m2)), S(e) && a(o.get(ye2)));
          break;
        case "delete":
          c || (a(o.get(m2)), S(e) && a(o.get(ye2)));
          break;
        case "set":
          S(e) && a(o.get(m2));
          break;
      }
    }
    Ve2();
  }
  function M2(e) {
    let t2 = p2(e);
    return t2 === e ? t2 : (w2(t2, "iterate", J2), y2(e) ? t2 : t2.map(D2));
  }
  function Ce2(e) {
    return w2(e = p2(e), "iterate", J2), e;
  }
  function b2(e, t2) {
    return V2(e) ? Q2(W2(e) ? D2(t2) : t2) : D2(t2);
  }
  var ht = { __proto__: null, [Symbol.iterator]() {
    return _e2(this, Symbol.iterator, (e) => b2(this, e));
  }, concat(...e) {
    return M2(this).concat(...e.map((t2) => d(t2) ? M2(t2) : t2));
  }, entries() {
    return _e2(this, "entries", (e) => (e[1] = b2(this, e[1]), e));
  }, every(e, t2) {
    return R2(this, "every", e, t2, void 0, arguments);
  }, filter(e, t2) {
    return R2(this, "filter", e, t2, (s) => s.map((n2) => b2(this, n2)), arguments);
  }, find(e, t2) {
    return R2(this, "find", e, t2, (s) => b2(this, s), arguments);
  }, findIndex(e, t2) {
    return R2(this, "findIndex", e, t2, void 0, arguments);
  }, findLast(e, t2) {
    return R2(this, "findLast", e, t2, (s) => b2(this, s), arguments);
  }, findLastIndex(e, t2) {
    return R2(this, "findLastIndex", e, t2, void 0, arguments);
  }, forEach(e, t2) {
    return R2(this, "forEach", e, t2, void 0, arguments);
  }, includes(...e) {
    return ve2(this, "includes", e);
  }, indexOf(...e) {
    return ve2(this, "indexOf", e);
  }, join(e) {
    return M2(this).join(e);
  }, lastIndexOf(...e) {
    return ve2(this, "lastIndexOf", e);
  }, map(e, t2) {
    return R2(this, "map", e, t2, void 0, arguments);
  }, pop() {
    return $2(this, "pop");
  }, push(...e) {
    return $2(this, "push", e);
  }, reduce(e, ...t2) {
    return Me(this, "reduce", e, t2);
  }, reduceRight(e, ...t2) {
    return Me(this, "reduceRight", e, t2);
  }, shift() {
    return $2(this, "shift");
  }, some(e, t2) {
    return R2(this, "some", e, t2, void 0, arguments);
  }, splice(...e) {
    return $2(this, "splice", e);
  }, toReversed() {
    return M2(this).toReversed();
  }, toSorted(e) {
    return M2(this).toSorted(e);
  }, toSpliced(...e) {
    return M2(this).toSpliced(...e);
  }, unshift(...e) {
    return $2(this, "unshift", e);
  }, values() {
    return _e2(this, "values", (e) => b2(this, e));
  } };
  function _e2(e, t2, s) {
    let n2 = Ce2(e), i = n2[t2]();
    return n2 !== e && !y2(e) && (i._next = i.next, i.next = () => {
      let r2 = i._next();
      return r2.done || (r2.value = s(r2.value)), r2;
    }), i;
  }
  var pt = Array.prototype;
  function R2(e, t2, s, n2, i, r2) {
    let o = Ce2(e), a = o !== e && !y2(e), c = o[t2];
    if (c !== pt[t2]) {
      let f2 = c.apply(e, r2);
      return a ? D2(f2) : f2;
    }
    let u = s;
    o !== e && (a ? u = function(f2, _) {
      return s.call(this, b2(e, f2), _, e);
    } : s.length > 2 && (u = function(f2, _) {
      return s.call(this, f2, _, e);
    }));
    let d2 = c.call(o, u, n2);
    return a && i ? i(d2) : d2;
  }
  function Me(e, t2, s, n2) {
    let i = Ce2(e), r2 = i !== e && !y2(e), o = s, a = false;
    i !== e && (r2 ? (a = n2.length === 0, o = function(u, d2, f2) {
      return a && (a = false, u = b2(e, u)), s.call(this, u, b2(e, d2), f2, e);
    }) : s.length > 3 && (o = function(u, d2, f2) {
      return s.call(this, u, d2, f2, e);
    }));
    let c = i[t2](o, ...n2);
    return a ? b2(e, c) : c;
  }
  function ve2(e, t2, s) {
    let n2 = p2(e);
    w2(n2, "iterate", J2);
    let i = n2[t2](...s);
    return (i === -1 || i === false) && qe(s[0]) ? (s[0] = p2(s[0]), n2[t2](...s)) : i;
  }
  function $2(e, t2, s = []) {
    Ue(), Ae2();
    let n2 = p2(e)[t2].apply(e, s);
    return Ve2(), $e(), n2;
  }
  var dt = l("__proto__,__v_isRef,__isVue");
  var Ye2 = new Set(Object.getOwnPropertyNames(Symbol).filter((e) => e !== "arguments" && e !== "caller").map((e) => Symbol[e]).filter(y));
  function _t(e) {
    y(e) || (e = String(e));
    let t2 = p2(this);
    return w2(t2, "has", e), t2.hasOwnProperty(e);
  }
  var oe2 = class {
    constructor(t2 = false, s = false) {
      this._isReadonly = t2, this._isShallow = s;
    }
    get(t2, s, n2) {
      if (s === "__v_skip") return t2.__v_skip;
      let i = this._isReadonly, r2 = this._isShallow;
      if (s === "__v_isReactive") return !i;
      if (s === "__v_isReadonly") return i;
      if (s === "__v_isShallow") return r2;
      if (s === "__v_raw") return n2 === (i ? r2 ? Je2 : Be2 : r2 ? ze : Fe).get(t2) || Object.getPrototypeOf(t2) === Object.getPrototypeOf(n2) ? t2 : void 0;
      let o = d(t2);
      if (!i) {
        let c;
        if (o && (c = ht[s])) return c;
        if (s === "hasOwnProperty") return _t;
      }
      let a = Reflect.get(t2, s, g2(t2) ? t2 : n2);
      if ((y(s) ? Ye2.has(s) : dt(s)) || (i || w2(t2, "get", s), r2)) return a;
      if (g2(a)) {
        let c = o && Ee(s) ? a : a.value;
        return i && f(c) ? be2(c) : c;
      }
      return f(a) ? i ? be2(a) : Qe(a) : a;
    }
  };
  var ae2 = class extends oe2 {
    constructor(t2 = false) {
      super(false, t2);
    }
    set(t2, s, n2, i) {
      let r2 = t2[s], o = d(t2) && Ee(s);
      if (!this._isShallow) {
        let u = V2(r2);
        if (!y2(n2) && !V2(n2) && (r2 = p2(r2), n2 = p2(n2)), !o && g2(r2) && !g2(n2)) return u || (r2.value = n2), true;
      }
      let a = o ? Number(s) < t2.length : ue(t2, s), c = Reflect.set(t2, s, n2, g2(t2) ? t2 : i);
      return t2 === p2(i) && c && (a ? Ne(n2, r2) && x(t2, "set", s, n2, r2) : x(t2, "add", s, n2)), c;
    }
    deleteProperty(t2, s) {
      let n2 = ue(t2, s), i = t2[s], r2 = Reflect.deleteProperty(t2, s);
      return r2 && n2 && x(t2, "delete", s, void 0, i), r2;
    }
    has(t2, s) {
      let n2 = Reflect.has(t2, s);
      return (!y(s) || !Ye2.has(s)) && w2(t2, "has", s), n2;
    }
    ownKeys(t2) {
      return w2(t2, "iterate", d(t2) ? "length" : m2), Reflect.ownKeys(t2);
    }
  };
  var ce2 = class extends oe2 {
    constructor(t2 = false) {
      super(true, t2);
    }
    set(t2, s) {
      return true;
    }
    deleteProperty(t2, s) {
      return true;
    }
  };
  var vt = new ae2();
  var gt = new ce2();
  var wt = new ae2(true);
  var Et = new ce2(true);
  var Se2 = (e) => e;
  var te = (e) => Reflect.getPrototypeOf(e);
  function yt(e, t2, s) {
    return function(...n2) {
      let i = this.__v_raw, r2 = p2(i), o = S(r2), a = e === "entries" || e === Symbol.iterator && o, c = e === "keys" && o, u = i[e](...n2), d2 = s ? Se2 : t2 ? Q2 : D2;
      return !t2 && w2(r2, "iterate", c ? ye2 : m2), de(Object.create(u), { next() {
        let { value: f2, done: _ } = u.next();
        return _ ? { value: f2, done: _ } : { value: a ? [d2(f2[0]), d2(f2[1])] : d2(f2), done: _ };
      } });
    };
  }
  function se2(e) {
    return function(...t2) {
      return e === "delete" ? false : e === "clear" ? void 0 : this;
    };
  }
  function St(e, t2) {
    let s = { get(i) {
      let r2 = this.__v_raw, o = p2(r2), a = p2(i);
      e || (Ne(i, a) && w2(o, "get", i), w2(o, "get", a));
      let { has: c } = te(o), u = t2 ? Se2 : e ? Q2 : D2;
      if (c.call(o, i)) return u(r2.get(i));
      if (c.call(o, a)) return u(r2.get(a));
      r2 !== o && r2.get(i);
    }, get size() {
      let i = this.__v_raw;
      return !e && w2(p2(i), "iterate", m2), i.size;
    }, has(i) {
      let r2 = this.__v_raw, o = p2(r2), a = p2(i);
      return e || (Ne(i, a) && w2(o, "has", i), w2(o, "has", a)), i === a ? r2.has(i) : r2.has(i) || r2.has(a);
    }, forEach(i, r2) {
      let o = this, a = o.__v_raw, c = p2(a), u = t2 ? Se2 : e ? Q2 : D2;
      return !e && w2(c, "iterate", m2), a.forEach((d2, f2) => i.call(r2, u(d2), u(f2), o));
    } };
    return de(s, e ? { add: se2("add"), set: se2("set"), delete: se2("delete"), clear: se2("clear") } : { add(i) {
      let r2 = p2(this), o = te(r2), a = p2(i), c = !t2 && !y2(i) && !V2(i) ? a : i;
      return o.has.call(r2, c) || Ne(i, c) && o.has.call(r2, i) || Ne(a, c) && o.has.call(r2, a) || (r2.add(c), x(r2, "add", c, c)), this;
    }, set(i, r2) {
      !t2 && !y2(r2) && !V2(r2) && (r2 = p2(r2));
      let o = p2(this), { has: a, get: c } = te(o), u = a.call(o, i);
      u || (i = p2(i), u = a.call(o, i));
      let d2 = c.call(o, i);
      return o.set(i, r2), u ? Ne(r2, d2) && x(o, "set", i, r2, d2) : x(o, "add", i, r2), this;
    }, delete(i) {
      let r2 = p2(this), { has: o, get: a } = te(r2), c = o.call(r2, i);
      c || (i = p2(i), c = o.call(r2, i));
      let u = a ? a.call(r2, i) : void 0, d2 = r2.delete(i);
      return c && x(r2, "delete", i, void 0, u), d2;
    }, clear() {
      let i = p2(this), r2 = i.size !== 0, o = void 0, a = i.clear();
      return r2 && x(i, "clear", void 0, void 0, o), a;
    } }), ["keys", "values", "entries", Symbol.iterator].forEach((i) => {
      s[i] = yt(i, e, t2);
    }), s;
  }
  function ue2(e, t2) {
    let s = St(e, t2);
    return (n2, i, r2) => i === "__v_isReactive" ? !e : i === "__v_isReadonly" ? e : i === "__v_raw" ? n2 : Reflect.get(ue(s, i) && i in n2 ? s : n2, i, r2);
  }
  var bt = { get: ue2(false, false) };
  var Nt = { get: ue2(false, true) };
  var Rt = { get: ue2(true, false) };
  var Tt = { get: ue2(true, true) };
  var Fe = /* @__PURE__ */ new WeakMap();
  var ze = /* @__PURE__ */ new WeakMap();
  var Be2 = /* @__PURE__ */ new WeakMap();
  var Je2 = /* @__PURE__ */ new WeakMap();
  function Dt(e) {
    switch (e) {
      case "Object":
      case "Array":
        return 1;
      case "Map":
      case "Set":
      case "WeakMap":
      case "WeakSet":
        return 2;
      default:
        return 0;
    }
  }
  function Qe(e) {
    return V2(e) ? e : he(e, false, vt, bt, Fe);
  }
  function Kt(e) {
    return he(e, false, wt, Nt, ze);
  }
  function be2(e) {
    return he(e, true, gt, Rt, Be2);
  }
  function he(e, t2, s, n2, i) {
    if (!f(e) || e.__v_raw && !(t2 && e.__v_isReactive) || e.__v_skip || !Object.isExtensible(e)) return e;
    let r2 = i.get(e);
    if (r2) return r2;
    let o = Dt(ye(e));
    if (o === 0) return e;
    let a = new Proxy(e, o === 2 ? n2 : s);
    return i.set(e, a), a;
  }
  function W2(e) {
    return V2(e) ? W2(e.__v_raw) : !!(e && e.__v_isReactive);
  }
  function V2(e) {
    return !!(e && e.__v_isReadonly);
  }
  function y2(e) {
    return !!(e && e.__v_isShallow);
  }
  function qe(e) {
    return e ? !!e.__v_raw : false;
  }
  function p2(e) {
    let t2 = e && e.__v_raw;
    return t2 ? p2(t2) : e;
  }
  function jt(e) {
    return !ue(e, "__v_skip") && Object.isExtensible(e) && xe(e, "__v_skip", true), e;
  }
  var D2 = (e) => f(e) ? Qe(e) : e;
  var Q2 = (e) => f(e) ? be2(e) : e;
  function g2(e) {
    return e ? e.__v_isRef === true : false;
  }
  function Ot(e) {
    return Xe2(e, false);
  }
  function Xe2(e, t2) {
    return g2(e) ? e : new Ne2(e, t2);
  }
  var Ne2 = class {
    constructor(t2, s) {
      this.dep = new K2(), this.__v_isRef = true, this.__v_isShallow = false, this._rawValue = s ? t2 : p2(t2), this._value = s ? t2 : D2(t2), this.__v_isShallow = s;
    }
    get value() {
      return this.dep.track(), this._value;
    }
    set value(t2) {
      let s = this._rawValue, n2 = this.__v_isShallow || y2(t2) || V2(t2);
      t2 = n2 ? t2 : p2(t2), Ne(t2, s) && (this._rawValue = t2, this._value = n2 ? t2 : D2(t2), this.dep.trigger());
    }
  };
  function Le(e) {
    return g2(e) ? e.value : e;
  }
  var xt = { get: (e, t2, s) => t2 === "__v_raw" ? e : Le(Reflect.get(e, t2, s)), set: (e, t2, s, n2) => {
    let i = e[t2];
    return g2(i) && !g2(s) ? (i.value = s, true) : Reflect.set(e, t2, s, n2);
  } };
  function Yt(e) {
    return W2(e) ? e : new Proxy(e, xt);
  }
  var Oe2 = class {
    constructor(t2, s, n2) {
      this.fn = t2, this.setter = s, this._value = void 0, this.dep = new K2(this), this.__v_isRef = true, this.deps = void 0, this.depsTail = void 0, this.flags = 16, this.globalVersion = B2 - 1, this.next = void 0, this.effect = this, this.__v_isReadonly = !s, this.isSSR = n2;
    }
    notify() {
      if (this.flags |= 16, !(this.flags & 8) && h !== this) return ke2(this, true), true;
    }
    get value() {
      let t2 = this.dep.track();
      return je(this), t2 && (t2.version = this.dep.version), this._value;
    }
    set value(t2) {
      this.setter && this.setter(t2);
    }
  };
  function Jt(e, t2, s = false) {
    let n2, i;
    return g(e) ? n2 = e : (n2 = e.get, i = e.set), new Oe2(n2, i, s);
  }
  var ie2 = {};
  var fe2 = /* @__PURE__ */ new WeakMap();
  var O;
  function At(e, t2 = false, s = O) {
    if (s) {
      let n2 = fe2.get(s);
      n2 || fe2.set(s, n2 = []), n2.push(e);
    }
  }
  function ts(e, t2, s = se) {
    let { immediate: n2, deep: i, once: r2, scheduler: o, augmentJob: a, call: c } = s, u = (l2) => {
      (s.onWarn || ct)("Invalid watch source: ", l2, "A watch source can only be a getter/effect function, a ref, a reactive object, or an array of these types.");
    }, d2 = (l2) => i ? l2 : y2(l2) || i === false || i === 0 ? A2(l2, 1) : A2(l2), f2, _, H2, q2, X2 = false, Z3 = false;
    if (g2(e) ? (_ = () => e.value, X2 = y2(e)) : W2(e) ? (_ = () => d2(e), X2 = true) : d(e) ? (Z3 = true, X2 = e.some((l2) => W2(l2) || y2(l2)), _ = () => e.map((l2) => {
      if (g2(l2)) return l2.value;
      if (W2(l2)) return d2(l2);
      if (g(l2)) return c ? c(l2, 2) : l2();
    })) : g(e) ? t2 ? _ = c ? () => c(e, 2) : e : _ = () => {
      if (H2) {
        Ue();
        try {
          H2();
        } finally {
          $e();
        }
      }
      let l2 = O;
      O = f2;
      try {
        return c ? c(e, 3, [q2]) : e(q2);
      } finally {
        O = l2;
      }
    } : _ = ce, t2 && i) {
      let l2 = _, S3 = i === true ? 1 / 0 : i;
      _ = () => A2(l2(), S3);
    }
    let pe3 = ft(), P2 = () => {
      f2.stop(), pe3 && pe3.active && me(pe3.effects, f2);
    };
    if (r2 && t2) {
      let l2 = t2;
      t2 = (...S3) => {
        let U2 = l2(...S3);
        return P2(), U2;
      };
    }
    let I2 = Z3 ? new Array(e.length).fill(ie2) : ie2, j3 = (l2) => {
      if (!(!(f2.flags & 1) || !f2.dirty && !l2)) if (t2) {
        let S3 = f2.run();
        if (l2 || i || X2 || (Z3 ? S3.some((U2, ee) => Ne(U2, I2[ee])) : Ne(S3, I2))) {
          H2 && H2();
          let U2 = O;
          O = f2;
          try {
            let ee = [S3, I2 === ie2 ? void 0 : Z3 && I2[0] === ie2 ? [] : I2, q2];
            I2 = S3, c ? c(t2, 3, ee) : t2(...ee);
          } finally {
            O = U2;
          }
        }
      } else f2.run();
    };
    return a && a(j3), f2 = new k2(_), f2.scheduler = o ? () => o(j3, false) : j3, q2 = (l2) => At(l2, false, f2), H2 = f2.onStop = () => {
      let l2 = fe2.get(f2);
      if (l2) {
        if (c) c(l2, 4);
        else for (let S3 of l2) S3();
        fe2.delete(f2);
      }
    }, t2 ? n2 ? j3(true) : I2 = f2.run() : o ? o(j3.bind(null, true), true) : f2.run(), P2.pause = f2.pause.bind(f2), P2.resume = f2.resume.bind(f2), P2.stop = P2, P2;
  }
  function A2(e, t2 = 1 / 0, s) {
    if (t2 <= 0 || !f(e) || e.__v_skip || (s = s || /* @__PURE__ */ new Map(), (s.get(e) || 0) >= t2)) return e;
    if (s.set(e, t2), t2--, g2(e)) A2(e.value, t2, s);
    else if (d(e)) for (let n2 = 0; n2 < e.length; n2++) A2(e[n2], t2, s);
    else if (N(e) || S(e)) e.forEach((n2) => {
      A2(n2, t2, s);
    });
    else if (z(e)) {
      for (let n2 in e) A2(e[n2], t2, s);
      for (let n2 of Object.getOwnPropertySymbols(e)) Object.prototype.propertyIsEnumerable.call(e, n2) && A2(e[n2], t2, s);
    }
    return e;
  }

  // choysum-esm:https://esm.sh/@vue/runtime-core@3.5.42/es2020/runtime-core.mjs?target=es2020
  function Et2(e, t2, n2, r2) {
    try {
      return r2 ? e(...r2) : e();
    } catch (o) {
      yt2(o, t2, n2);
    }
  }
  function Ie2(e, t2, n2, r2) {
    if (g(e)) {
      let o = Et2(e, t2, n2, r2);
      return o && ge(o) && o.catch((s) => {
        yt2(s, t2, n2);
      }), o;
    }
    if (d(e)) {
      let o = [];
      for (let s = 0; s < e.length; s++) o.push(Ie2(e[s], t2, n2, r2));
      return o;
    }
  }
  function yt2(e, t2, n2, r2 = true) {
    let o = t2 ? t2.vnode : null, { errorHandler: s, throwUnhandledErrorInProduction: i } = t2 && t2.appContext.config || se;
    if (t2) {
      let l2 = t2.parent, a = t2.proxy, h2 = `https://vuejs.org/error-reference/#runtime-${n2}`;
      for (; l2; ) {
        let f2 = l2.ec;
        if (f2) {
          for (let u = 0; u < f2.length; u++) if (f2[u](e, a, h2) === false) return;
        }
        l2 = l2.parent;
      }
      if (s) {
        Ue(), Et2(s, null, 10, [e, a, h2]), $e();
        return;
      }
    }
    gs(e, n2, o, r2, i);
  }
  function gs(e, t2, n2, r2 = true, o = false) {
    if (o) throw e;
    console.error(e);
  }
  var ye3 = [];
  var Pe2 = -1;
  var pt2 = [];
  var Je3 = null;
  var at = 0;
  var Wr = Promise.resolve();
  var Qt2 = null;
  function ms(e) {
    let t2 = Qt2 || Wr;
    return e ? t2.then(this ? e.bind(this) : e) : t2;
  }
  function Es(e) {
    let t2 = Pe2 + 1, n2 = ye3.length;
    for (; t2 < n2; ) {
      let r2 = t2 + n2 >>> 1, o = ye3[r2], s = At2(o);
      s < e || s === e && o.flags & 2 ? t2 = r2 + 1 : n2 = r2;
    }
    return t2;
  }
  function Xn(e) {
    if (!(e.flags & 1)) {
      let t2 = At2(e), n2 = ye3[ye3.length - 1];
      !n2 || !(e.flags & 2) && t2 >= At2(n2) ? ye3.push(e) : ye3.splice(Es(t2), 0, e), e.flags |= 1, Kr();
    }
  }
  function Kr() {
    Qt2 || (Qt2 = Wr.then(Yr));
  }
  function $n(e) {
    if (!d(e)) Je3 && e.id === -1 ? Je3.splice(at + 1, 0, e) : e.flags & 1 || (pt2.push(e), e.flags |= 1);
    else for (let t2 = 0; t2 < e.length; t2++) pt2.push(e[t2]);
    Kr();
  }
  function Er(e, t2, n2 = Pe2 + 1) {
    for (; n2 < ye3.length; n2++) {
      let r2 = ye3[n2];
      if (r2 && r2.flags & 2) {
        if (e && r2.id !== e.uid) continue;
        ye3.splice(n2, 1), n2--, r2.flags & 4 && (r2.flags &= -2), r2(), r2.flags & 4 || (r2.flags &= -2);
      }
    }
  }
  function Zt(e) {
    if (pt2.length) {
      let t2 = [...new Set(pt2)].sort((n2, r2) => At2(n2) - At2(r2));
      if (pt2.length = 0, Je3) {
        for (let n2 = 0; n2 < t2.length; n2++) Je3.push(t2[n2]);
        return;
      }
      for (Je3 = t2, at = 0; at < Je3.length; at++) {
        let n2 = Je3[at];
        n2.flags & 4 && (n2.flags &= -2), n2.flags & 8 || n2(), n2.flags &= -2;
      }
      Je3 = null, at = 0;
    }
  }
  var At2 = (e) => e.id == null ? e.flags & 2 ? -1 : 1 / 0 : e.id;
  function Yr(e) {
    let t2 = ce;
    try {
      for (Pe2 = 0; Pe2 < ye3.length; Pe2++) {
        let n2 = ye3[Pe2];
        n2 && !(n2.flags & 8) && (n2.flags & 4 && (n2.flags &= -2), Et2(n2, n2.i, n2.i ? 15 : 14), n2.flags & 4 || (n2.flags &= -2));
      }
    } finally {
      for (; Pe2 < ye3.length; Pe2++) {
        let n2 = ye3[Pe2];
        n2 && (n2.flags &= -2);
      }
      Pe2 = -1, ye3.length = 0, Zt(e), Qt2 = null, (ye3.length || pt2.length) && Yr(e);
    }
  }
  var ys = false;
  var Te2;
  var Dt2 = [];
  var kn = false;
  function dn(e, ...t2) {
    Te2 ? Te2.emit(e, ...t2) : kn || Dt2.push({ event: e, args: t2 });
  }
  var An = Zn("component:added");
  var qr = Zn("component:updated");
  var Os = Zn("component:removed");
  function Zn(e) {
    return (t2) => {
      dn(e, t2.appContext.app, t2.uid, t2.parent ? t2.parent.uid : void 0, t2);
    };
  }
  var de3 = null;
  var hn = null;
  function Pt2(e) {
    let t2 = de3;
    return de3 = e, hn = e && e.type.__scopeId || null, t2;
  }
  function Jr(e, t2 = de3, n2) {
    if (!t2 || e._n) return e;
    let r2 = (...o) => {
      r2._d && ln(-1);
      let s = Pt2(t2), i = Le2.length, l2;
      try {
        l2 = e(...o);
      } finally {
        for (let a = Le2.length; a > i; a--) yn();
        Pt2(s), r2._d && ln(1);
      }
      return false, l2;
    };
    return r2._n = true, r2._c = true, r2._d = true, r2;
  }
  function Re(e, t2, n2, r2) {
    let o = e.dirs, s = t2 && t2.dirs;
    for (let i = 0; i < o.length; i++) {
      let l2 = o[i];
      s && (l2.oldValue = s[i].value);
      let a = l2.dir[r2];
      a && (Ue(), Ie2(a, n2, 8, [e.el, l2, e, t2]), $e());
    }
  }
  function Vs(e, t2) {
    if (pe2) {
      let n2 = pe2.provides, r2 = pe2.parent && pe2.parent.provides;
      r2 === n2 && (n2 = pe2.provides = Object.create(r2)), n2[e] = t2;
    }
  }
  function qt2(e, t2, n2 = false) {
    let r2 = Me2();
    if (r2 || st) {
      let o = st ? st._context.provides : r2 ? r2.parent == null || r2.ce ? r2.vnode.appContext && r2.vnode.appContext.provides : r2.parent.provides : void 0;
      if (o && e in o) return o[e];
      if (arguments.length > 1) return n2 && g(t2) ? t2.call(r2 && r2.proxy) : t2;
    }
  }
  var ws = /* @__PURE__ */ Symbol.for("v-scx");
  var xs = () => {
    {
      let e = qt2(ws);
      return e;
    }
  };
  function Jt2(e, t2, n2) {
    return Mt2(e, t2, n2);
  }
  function Mt2(e, t2, n2 = se) {
    let { immediate: r2, deep: o, flush: s, once: i } = n2, l2 = de({}, n2), a = t2 && r2 || !t2 && s !== "post", h2;
    if (ct2) {
      if (s === "sync") {
        let y4 = xs();
        h2 = y4.__watcherHandles || (y4.__watcherHandles = []);
      } else if (!a) {
        let y4 = () => {
        };
        return y4.stop = ce, y4.resume = ce, y4.pause = ce, y4;
      }
    }
    let f2 = pe2;
    l2.call = (y4, v4, N2) => Ie2(y4, f2, v4, N2);
    let u = false;
    s === "post" ? l2.scheduler = (y4) => {
      ce3(y4, f2 && f2.suspense);
    } : s !== "sync" && (u = true, l2.scheduler = (y4, v4) => {
      v4 ? y4() : Xn(y4);
    }), l2.augmentJob = (y4) => {
      t2 && (y4.flags |= 4), u && (y4.flags |= 2, f2 && (y4.id = f2.uid, y4.i = f2));
    };
    let E3 = ts(e, t2, l2);
    return ct2 && (h2 ? h2.push(E3) : a && E3()), E3;
  }
  function Cs(e, t2, n2) {
    let r2 = this.proxy, o = p(e) ? e.includes(".") ? Gr(r2, e) : () => r2[e] : e.bind(r2, r2), s;
    g(t2) ? s = t2 : (s = t2.handler, n2 = t2);
    let i = Nt2(this), l2 = Mt2(o, s.bind(r2), n2);
    return i(), l2;
  }
  function Gr(e, t2) {
    let n2 = t2.split(".");
    return () => {
      let r2 = e;
      for (let o = 0; o < n2.length && r2; o++) r2 = r2[n2[o]];
      return r2;
    };
  }
  var Xr = /* @__PURE__ */ Symbol("_vte");
  var _n = (e) => e.__isTeleport;
  var Ve3 = /* @__PURE__ */ Symbol("_leaveCb");
  var bt2 = /* @__PURE__ */ Symbol("_enterCb");
  function Ps() {
    let e = { isMounted: false, isLeaving: false, isUnmounting: false, leavingVNodes: /* @__PURE__ */ new Map() };
    return tr(() => {
      e.isMounted = true;
    }), nr(() => {
      e.isUnmounting = true;
    }), e;
  }
  var De = [Function, Array];
  var Ss = { mode: String, appear: Boolean, persisted: Boolean, onBeforeEnter: De, onEnter: De, onAfterEnter: De, onEnterCancelled: De, onBeforeLeave: De, onLeave: De, onAfterLeave: De, onLeaveCancelled: De, onBeforeAppear: De, onAppear: De, onAfterAppear: De, onAppearCancelled: De };
  var Qr = (e) => {
    let t2 = e.subTree;
    return t2.component ? Qr(t2.component) : t2;
  };
  var Rs = { name: "BaseTransition", props: Ss, setup(e, { slots: t2 }) {
    let n2 = Me2(), r2 = Ps();
    return () => {
      let o = t2.default && eo(t2.default(), true), s = o && o.length ? Zr(o) : n2.subTree ? Ui() : void 0;
      if (!s) return;
      let i = p2(e), { mode: l2 } = i;
      if (r2.isLeaving) return Vn(s);
      let a = zt2(s);
      if (!a) return Vn(s);
      let h2 = Rn(a, i, r2, n2, (u) => h2 = u);
      a.type !== oe3 && gt2(a, h2);
      let f2 = n2.subTree && zt2(n2.subTree);
      if (f2 && f2.type !== oe3 && !Ce3(f2, a) && Qr(n2).type !== oe3) {
        let u = Rn(f2, i, r2, n2);
        if (gt2(f2, u), l2 === "out-in" && a.type !== oe3) return r2.isLeaving = true, u.afterLeave = () => {
          r2.isLeaving = false, n2.job.flags & 8 || n2.update(), delete u.afterLeave, f2 = void 0;
        }, Vn(s);
        l2 === "in-out" && a.type !== oe3 ? u.delayLeave = (E3, y4, v4) => {
          let N2 = zr(r2, f2);
          N2[String(f2.key)] = f2, E3[Ve3] = () => {
            y4(), E3[Ve3] = void 0, delete h2.delayedLeave, f2 = void 0;
          }, h2.delayedLeave = () => {
            v4(), delete h2.delayedLeave, f2 = void 0;
          };
        } : f2 = void 0;
      } else f2 && (f2 = void 0);
      return s;
    };
  } };
  function Zr(e) {
    let t2 = e[0];
    if (e.length > 1) {
      let n2 = false;
      for (let r2 of e) if (r2.type !== oe3) {
        t2 = r2, n2 = true;
        break;
      }
    }
    return t2;
  }
  var _l = Rs;
  function zr(e, t2) {
    let { leavingVNodes: n2 } = e, r2 = n2.get(t2.type);
    return r2 || (r2 = /* @__PURE__ */ Object.create(null), n2.set(t2.type, r2)), r2;
  }
  function Rn(e, t2, n2, r2, o) {
    let { appear: s, mode: i, persisted: l2 = false, onBeforeEnter: a, onEnter: h2, onAfterEnter: f2, onEnterCancelled: u, onBeforeLeave: E3, onLeave: y4, onAfterLeave: v4, onLeaveCancelled: N2, onBeforeAppear: M4, onAppear: U2, onAfterAppear: V3, onAppearCancelled: d2 } = t2, g4 = String(e.key), _ = zr(n2, e), $3 = (w4, S3) => {
      w4 && Ie2(w4, r2, 9, S3);
    }, C2 = (w4, S3) => {
      let P2 = S3[1];
      $3(w4, S3), d(w4) ? w4.every((W4) => W4.length <= 1) && P2() : w4.length <= 1 && P2();
    }, I2 = { mode: i, persisted: l2, beforeEnter(w4) {
      let S3 = a;
      if (!n2.isMounted) if (s) S3 = M4 || a;
      else return;
      w4[Ve3] && w4[Ve3](true);
      let P2 = _[g4];
      P2 && Ce3(e, P2) && P2.el[Ve3] && P2.el[Ve3](), $3(S3, [w4]);
    }, enter(w4) {
      if (!ys && _[g4] === e) return;
      let S3 = h2, P2 = f2, W4 = u;
      if (!n2.isMounted) if (s) S3 = U2 || h2, P2 = V3 || f2, W4 = d2 || u;
      else return;
      let J4 = false;
      w4[bt2] = (ee) => {
        J4 || (J4 = true, ee ? $3(W4, [w4]) : $3(P2, [w4]), I2.delayedLeave && I2.delayedLeave(), w4[bt2] = void 0);
      };
      let G = w4[bt2].bind(null, false);
      S3 ? C2(S3, [w4, G]) : G();
    }, leave(w4, S3) {
      let P2 = String(e.key);
      if (w4[bt2] && w4[bt2](true), n2.isUnmounting) return S3();
      $3(E3, [w4]);
      let W4 = false;
      w4[Ve3] = (G) => {
        W4 || (W4 = true, S3(), G ? $3(N2, [w4]) : $3(v4, [w4]), w4[Ve3] = void 0, _[P2] === e && delete _[P2]);
      };
      let J4 = w4[Ve3].bind(null, false);
      _[P2] = e, y4 ? C2(y4, [w4, J4]) : J4();
    }, clone(w4) {
      let S3 = Rn(w4, t2, n2, r2, o);
      return o && o(S3), S3;
    } };
    return I2;
  }
  function Vn(e) {
    if (Ft2(e)) return e = We3(e), e.children = null, e;
  }
  function zt2(e) {
    if (!Ft2(e)) return _n(e.type) && e.children ? Zr(e.children) : e;
    if (e.component) return e.component.subTree;
    let { shapeFlag: t2, children: n2 } = e;
    if (n2) {
      if (t2 & 16) return n2[0];
      if (t2 & 32 && g(n2.default)) return n2.default();
    }
  }
  function gt2(e, t2) {
    if (e.shapeFlag & 6 && e.component) {
      e.transition = t2;
      let n2 = e.component.subTree;
      gt2(_n(n2.type) && zt2(n2) || n2, t2);
    } else e.shapeFlag & 128 ? (e.ssContent.transition = t2.clone(e.ssContent), e.ssFallback.transition = t2.clone(e.ssFallback)) : e.transition = t2;
  }
  function eo(e, t2 = false, n2) {
    let r2 = [], o = 0;
    for (let s = 0; s < e.length; s++) {
      let i = e[s], l2 = n2 == null ? i.key : String(n2) + String(i.key != null ? i.key : s);
      i.type === ge3 ? (i.patchFlag & 128 && o++, r2 = r2.concat(eo(i.children, t2, l2))) : (t2 || i.type !== oe3) && r2.push(l2 != null ? We3(i, { key: l2 }) : i);
    }
    if (o > 1) for (let s = 0; s < r2.length; s++) r2[s].patchFlag = -2;
    return r2;
  }
  function Is(e, t2) {
    return g(e) ? de({ name: e.name }, t2, { setup: e }) : e;
  }
  function zn(e) {
    e.ids = [e.ids[0] + e.ids[2]++ + "-", 0, 0];
  }
  function vr(e, t2) {
    let n2;
    return !!((n2 = Object.getOwnPropertyDescriptor(e, t2)) && !n2.configurable);
  }
  var en = /* @__PURE__ */ new WeakMap();
  function dt2(e, t2, n2, r2, o = false) {
    if (d(e)) {
      e.forEach((N2, M4) => dt2(N2, t2 && (d(t2) ? t2[M4] : t2), n2, r2, o));
      return;
    }
    if (Ue2(r2) && !o) {
      r2.shapeFlag & 512 && r2.type.__asyncResolved && r2.component.subTree.component && dt2(e, t2, n2, r2.component.subTree);
      return;
    }
    let s = r2.shapeFlag & 4 ? Ht2(r2.component) : r2.el, i = o ? null : s, { i: l2, r: a } = e, h2 = t2 && t2.r, f2 = l2.refs === se ? l2.refs = {} : l2.refs, u = l2.setupState, E3 = p2(u), y4 = u === se ? le : (N2) => vr(f2, N2) ? false : ue(E3, N2), v4 = (N2, M4) => !(M4 && vr(f2, M4));
    if (h2 != null && h2 !== a) {
      if (Or(t2), p(h2)) f2[h2] = null, y4(h2) && (u[h2] = null);
      else if (g2(h2)) {
        let N2 = t2;
        v4(h2, N2.k) && (h2.value = null), N2.k && (f2[N2.k] = null);
      }
    }
    if (g(a)) Et2(a, l2, 12, [i, f2]);
    else {
      let N2 = p(a), M4 = g2(a);
      if (N2 || M4) {
        let U2 = () => {
          if (e.f) {
            let V3 = N2 ? y4(a) ? u[a] : f2[a] : v4(a) || !e.k ? a.value : f2[e.k];
            if (o) d(V3) && me(V3, s);
            else if (d(V3)) V3.includes(s) || V3.push(s);
            else if (N2) f2[a] = [s], y4(a) && (u[a] = f2[a]);
            else {
              let d2 = [s];
              v4(a, e.k) && (a.value = d2), e.k && (f2[e.k] = d2);
            }
          } else N2 ? (f2[a] = i, y4(a) && (u[a] = i)) : M4 && (v4(a, e.k) && (a.value = i), e.k && (f2[e.k] = i));
        };
        if (i) {
          let V3 = () => {
            U2(), en.delete(e);
          };
          V3.id = -1, en.set(e, V3), ce3(V3, n2);
        } else Or(e), U2();
      }
    }
  }
  function Or(e) {
    let t2 = en.get(e);
    t2 && (t2.flags |= 8, en.delete(e));
  }
  var Js = _e().requestIdleCallback || ((e) => setTimeout(e, 1));
  var Gs = _e().cancelIdleCallback || ((e) => clearTimeout(e));
  var Ue2 = (e) => !!e.type.__asyncLoader;
  var Ft2 = (e) => e.type.__isKeepAlive;
  function zs(e, t2) {
    ro(e, "a", t2);
  }
  function ei(e, t2) {
    ro(e, "da", t2);
  }
  function ro(e, t2, n2 = pe2) {
    let r2 = e.__wdc || (e.__wdc = () => {
      let o = n2;
      for (; o; ) {
        if (o.isDeactivated) return;
        o = o.parent;
      }
      return e();
    });
    if (gn(t2, r2, n2), n2) {
      let o = n2.parent;
      for (; o && o.parent; ) Ft2(o.parent.vnode) && ti(r2, t2, n2, o), o = o.parent;
    }
  }
  function ti(e, t2, n2, r2) {
    let o = gn(t2, e, r2, true);
    rr(() => {
      me(r2[t2], o);
    }, n2);
  }
  function gn(e, t2, n2 = pe2, r2 = false) {
    if (n2) {
      let o = n2[e] || (n2[e] = []), s = t2.__weh || (t2.__weh = (...i) => {
        Ue();
        let l2 = Nt2(n2), a = Ie2(t2, n2, e, i);
        return l2(), $e(), a;
      });
      return r2 ? o.unshift(s) : o.push(s), s;
    }
  }
  var Ke3 = (e) => (t2, n2 = pe2) => {
    (!ct2 || e === "sp") && gn(e, (...r2) => t2(...r2), n2);
  };
  var ni = Ke3("bm");
  var tr = Ke3("m");
  var ri = Ke3("bu");
  var oo = Ke3("u");
  var nr = Ke3("bum");
  var rr = Ke3("um");
  var oi = Ke3("sp");
  var si = Ke3("rtg");
  var ii = Ke3("rtc");
  function li(e, t2 = pe2) {
    gn("ec", e, t2);
  }
  var so = /* @__PURE__ */ Symbol.for("v-ndc");
  var In = (e) => e ? Ao(e) ? Ht2(e) : In(e.parent) : null;
  var $t2 = de(/* @__PURE__ */ Object.create(null), { $: (e) => e, $el: (e) => e.vnode.el, $data: (e) => e.data, $props: (e) => e.props, $attrs: (e) => e.attrs, $slots: (e) => e.slots, $refs: (e) => e.refs, $parent: (e) => In(e.parent), $root: (e) => In(e.root), $host: (e) => e.ce, $emit: (e) => e.emit, $options: (e) => true ? lr(e) : e.type, $forceUpdate: (e) => e.f || (e.f = () => {
    Xn(e.update);
  }), $nextTick: (e) => e.n || (e.n = ms.bind(e.proxy)), $watch: (e) => true ? Cs.bind(e) : ce });
  var xn = (e, t2) => e !== se && !e.__isScriptSetup && ue(e, t2);
  var Mn = { get({ _: e }, t2) {
    if (t2 === "__v_skip") return true;
    let { ctx: n2, setupState: r2, data: o, props: s, accessCache: i, type: l2, appContext: a } = e;
    if (t2[0] !== "$") {
      let E3 = i[t2];
      if (E3 !== void 0) switch (E3) {
        case 1:
          return r2[t2];
        case 2:
          return o[t2];
        case 4:
          return n2[t2];
        case 3:
          return s[t2];
      }
      else {
        if (xn(r2, t2)) return i[t2] = 1, r2[t2];
        if (o !== se && ue(o, t2)) return i[t2] = 2, o[t2];
        if (ue(s, t2)) return i[t2] = 3, s[t2];
        if (n2 !== se && ue(n2, t2)) return i[t2] = 4, n2[t2];
        Fn && (i[t2] = 0);
      }
    }
    let h2 = $t2[t2], f2, u;
    if (h2) return t2 === "$attrs" && w2(e.attrs, "get", ""), h2(e);
    if ((f2 = l2.__cssModules) && (f2 = f2[t2])) return f2;
    if (n2 !== se && ue(n2, t2)) return i[t2] = 4, n2[t2];
    if (u = a.config.globalProperties, ue(u, t2)) return u[t2];
  }, set({ _: e }, t2, n2) {
    let { data: r2, setupState: o, ctx: s } = e;
    return xn(o, t2) ? (o[t2] = n2, true) : r2 !== se && ue(r2, t2) ? (r2[t2] = n2, true) : ue(e.props, t2) || t2[0] === "$" && t2.slice(1) in e ? false : (s[t2] = n2, true);
  }, has({ _: { data: e, setupState: t2, accessCache: n2, ctx: r2, appContext: o, props: s, type: i } }, l2) {
    let a;
    return !!(n2[l2] || e !== se && l2[0] !== "$" && ue(e, l2) || xn(t2, l2) || ue(s, l2) || ue(r2, l2) || ue($t2, l2) || ue(o.config.globalProperties, l2) || (a = i.__cssModules) && a[l2]);
  }, defineProperty(e, t2, n2) {
    return n2.get != null ? e._.accessCache[t2] = 0 : ue(n2, "value") && this.set(e, t2, n2.value, null), Reflect.defineProperty(e, t2, n2);
  } };
  var ui = de({}, Mn, { get(e, t2) {
    if (t2 !== Symbol.unscopables) return Mn.get(e, t2, e);
  }, has(e, t2) {
    return t2[0] !== "_" && !v(t2);
  } });
  function St2(e) {
    return d(e) ? e.reduce((t2, n2) => (t2[n2] = null, t2), {}) : e;
  }
  var Fn = true;
  function ai(e) {
    let t2 = lr(e), n2 = e.proxy, r2 = e.ctx;
    Fn = false, t2.beforeCreate && Tr(t2.beforeCreate, e, "bc");
    let { data: o, computed: s, methods: i, watch: l2, provide: a, inject: h2, created: f2, beforeMount: u, mounted: E3, beforeUpdate: y4, updated: v4, activated: N2, deactivated: M4, beforeDestroy: U2, beforeUnmount: V3, destroyed: d2, unmounted: g4, render: _, renderTracked: $3, renderTriggered: C2, errorCaptured: I2, serverPrefetch: w4, expose: S3, inheritAttrs: P2, components: W4, directives: J4, filters: G } = t2;
    if (h2 && fi(h2, r2, null), i) for (let Y4 in i) {
      let B3 = i[Y4];
      g(B3) && (r2[Y4] = B3.bind(n2));
    }
    if (o) {
      let Y4 = o.call(n2, n2);
      f(Y4) && (e.data = Qe(Y4));
    }
    if (Fn = true, s) for (let Y4 in s) {
      let B3 = s[Y4], re2 = g(B3) ? B3.bind(n2, n2) : g(B3.get) ? B3.get.bind(n2, n2) : ce, te2 = !g(B3) && g(B3.set) ? B3.set.bind(n2) : ce, _e3 = Gi({ get: re2, set: te2 });
      Object.defineProperty(r2, Y4, { enumerable: true, configurable: true, get: () => _e3.value, set: (ie4) => _e3.value = ie4 });
    }
    if (l2) for (let Y4 in l2) lo(l2[Y4], r2, n2, Y4);
    if (a) {
      let Y4 = g(a) ? a.call(n2) : a;
      Reflect.ownKeys(Y4).forEach((B3) => {
        Vs(B3, Y4[B3]);
      });
    }
    f2 && Tr(f2, e, "c");
    function H2(Y4, B3) {
      d(B3) ? B3.forEach((re2) => Y4(re2.bind(n2))) : B3 && Y4(B3.bind(n2));
    }
    if (H2(ni, u), H2(tr, E3), H2(ri, y4), H2(oo, v4), H2(zs, N2), H2(ei, M4), H2(li, I2), H2(ii, $3), H2(si, C2), H2(nr, V3), H2(rr, g4), H2(oi, w4), d(S3)) if (S3.length) {
      let Y4 = e.exposed || (e.exposed = {});
      S3.forEach((B3) => {
        Object.defineProperty(Y4, B3, { get: () => n2[B3], set: (re2) => n2[B3] = re2, enumerable: true });
      });
    } else e.exposed || (e.exposed = {});
    _ && e.render === ce && (e.render = _), P2 != null && (e.inheritAttrs = P2), W4 && (e.components = W4), J4 && (e.directives = J4), w4 && zn(e);
  }
  function fi(e, t2, n2 = ce) {
    d(e) && (e = Hn(e));
    for (let r2 in e) {
      let o = e[r2], s;
      f(o) ? "default" in o ? s = qt2(o.from || r2, o.default, true) : s = qt2(o.from || r2) : s = qt2(o), g2(s) ? Object.defineProperty(t2, r2, { enumerable: true, configurable: true, get: () => s.value, set: (i) => s.value = i }) : t2[r2] = s;
    }
  }
  function Tr(e, t2, n2) {
    Ie2(d(e) ? e.map((r2) => r2.bind(t2.proxy)) : e.bind(t2.proxy), t2, n2);
  }
  function lo(e, t2, n2, r2) {
    let o = r2.includes(".") ? Gr(n2, r2) : () => n2[r2];
    if (p(e)) {
      let s = t2[e];
      g(s) && Jt2(o, s);
    } else if (g(e)) Jt2(o, e.bind(n2));
    else if (f(e)) if (d(e)) e.forEach((s) => lo(s, t2, n2, r2));
    else {
      let s = g(e.handler) ? e.handler.bind(n2) : t2[e.handler];
      g(s) && Jt2(o, s, e);
    }
  }
  function lr(e) {
    let t2 = e.type, { mixins: n2, extends: r2 } = t2, { mixins: o, optionsCache: s, config: { optionMergeStrategies: i } } = e.appContext, l2 = s.get(t2), a;
    return l2 ? a = l2 : !o.length && !n2 && !r2 ? a = t2 : (a = {}, o.length && o.forEach((h2) => nn(a, h2, i, true)), nn(a, t2, i)), f(t2) && s.set(t2, a), a;
  }
  function nn(e, t2, n2, r2 = false) {
    let { mixins: o, extends: s } = t2;
    s && nn(e, s, n2, true), o && o.forEach((i) => nn(e, i, n2, true));
    for (let i in t2) if (!(r2 && i === "expose")) {
      let l2 = pi[i] || n2 && n2[i];
      e[i] = l2 ? l2(e[i], t2[i]) : t2[i];
    }
    return e;
  }
  var pi = { data: Cr, props: $r, emits: $r, methods: xt2, computed: xt2, beforeCreate: Ee3, created: Ee3, beforeMount: Ee3, mounted: Ee3, beforeUpdate: Ee3, updated: Ee3, beforeDestroy: Ee3, beforeUnmount: Ee3, destroyed: Ee3, unmounted: Ee3, activated: Ee3, deactivated: Ee3, errorCaptured: Ee3, serverPrefetch: Ee3, components: xt2, directives: xt2, watch: hi, provide: Cr, inject: di };
  function Cr(e, t2) {
    return t2 ? e ? function() {
      return de(g(e) ? e.call(this, this) : e, g(t2) ? t2.call(this, this) : t2);
    } : t2 : e;
  }
  function di(e, t2) {
    return xt2(Hn(e), Hn(t2));
  }
  function Hn(e) {
    if (d(e)) {
      let t2 = {};
      for (let n2 = 0; n2 < e.length; n2++) t2[e[n2]] = e[n2];
      return t2;
    }
    return e;
  }
  function Ee3(e, t2) {
    return e ? [...new Set([].concat(e, t2))] : t2;
  }
  function xt2(e, t2) {
    return e ? de(/* @__PURE__ */ Object.create(null), e, t2) : t2;
  }
  function $r(e, t2) {
    return e ? d(e) && d(t2) ? [.../* @__PURE__ */ new Set([...e, ...t2])] : de(/* @__PURE__ */ Object.create(null), St2(e), St2(t2 ?? {})) : t2;
  }
  function hi(e, t2) {
    if (!e) return t2;
    if (!t2) return e;
    let n2 = de(/* @__PURE__ */ Object.create(null), e);
    for (let r2 in t2) n2[r2] = Ee3(e[r2], t2[r2]);
    return n2;
  }
  function co() {
    return { app: null, config: { isNativeTag: le, performance: false, globalProperties: {}, optionMergeStrategies: {}, errorHandler: void 0, warnHandler: void 0, compilerOptions: {} }, mixins: [], components: {}, directives: {}, provides: /* @__PURE__ */ Object.create(null), optionsCache: /* @__PURE__ */ new WeakMap(), propsCache: /* @__PURE__ */ new WeakMap(), emitsCache: /* @__PURE__ */ new WeakMap() };
  }
  var _i = 0;
  function gi(e, t2) {
    return function(r2, o = null) {
      g(r2) || (r2 = de({}, r2)), o != null && !f(o) && (o = null);
      let s = co(), i = /* @__PURE__ */ new WeakSet(), l2 = [], a = false, h2 = s.app = { _uid: _i++, _component: r2, _props: o, _container: null, _context: s, _instance: null, version: Sr, get config() {
        return s.config;
      }, set config(f2) {
      }, use(f2, ...u) {
        return i.has(f2) || (f2 && g(f2.install) ? (i.add(f2), f2.install(h2, ...u)) : g(f2) && (i.add(f2), f2(h2, ...u))), h2;
      }, mixin(f2) {
        return s.mixins.includes(f2) || s.mixins.push(f2), h2;
      }, component(f2, u) {
        return u ? (s.components[f2] = u, h2) : s.components[f2];
      }, directive(f2, u) {
        return u ? (s.directives[f2] = u, h2) : s.directives[f2];
      }, mount(f2, u, E3) {
        if (!a) {
          let y4 = h2._ceVNode || se3(r2, o);
          return y4.appContext = s, E3 === true ? E3 = "svg" : E3 === false && (E3 = void 0), u && t2 ? t2(y4, f2) : e(y4, f2, E3), a = true, h2._container = f2, f2.__vue_app__ = h2, false, Ht2(y4.component);
        }
      }, onUnmount(f2) {
        l2.push(f2);
      }, unmount() {
        a && (Ie2(l2, h2._instance, 16), e(null, h2._container), false, delete h2._container.__vue_app__);
      }, provide(f2, u) {
        return s.provides[f2] = u, h2;
      }, runWithContext(f2) {
        let u = st;
        st = h2;
        try {
          return f2();
        } finally {
          st = u;
        }
      } };
      return h2;
    };
  }
  var st = null;
  var uo = (e, t2) => t2 === "modelValue" || t2 === "model-value" ? e.modelModifiers : e[`${t2}Modifiers`] || e[`${Ae(t2)}Modifiers`] || e[`${H(t2)}Modifiers`];
  function mi(e, t2, ...n2) {
    if (e.isUnmounted) return;
    let r2 = e.vnode.props || se, o = n2, s = t2.startsWith("update:"), i = s && uo(r2, t2.slice(7));
    i && (i.trim && (o = n2.map((f2) => p(f2) ? f2.trim() : f2)), i.number && (o = o.map(ke))), false;
    let l2, a = r2[l2 = Se(t2)] || r2[l2 = Se(Ae(t2))];
    !a && s && (a = r2[l2 = Se(H(t2))]), a && Ie2(a, e, 6, o);
    let h2 = r2[l2 + "Once"];
    if (h2) {
      if (!e.emitted) e.emitted = {};
      else if (e.emitted[l2]) return;
      e.emitted[l2] = true, Ie2(h2, e, 6, o);
    }
  }
  var Ei = /* @__PURE__ */ new WeakMap();
  function ao(e, t2, n2 = false) {
    let r2 = n2 ? Ei : t2.emitsCache, o = r2.get(e);
    if (o !== void 0) return o;
    let s = e.emits, i = {}, l2 = false;
    if (!g(e)) {
      let a = (h2) => {
        let f2 = ao(h2, t2, true);
        f2 && (l2 = true, de(i, f2));
      };
      !n2 && t2.mixins.length && t2.mixins.forEach(a), e.extends && a(e.extends), e.mixins && e.mixins.forEach(a);
    }
    return !s && !l2 ? (f(e) && r2.set(e, null), null) : (d(s) ? s.forEach((a) => i[a] = null) : de(i, s), f(e) && r2.set(e, i), i);
  }
  function mn(e, t2) {
    return !e || !pe(t2) ? false : (t2 = t2.slice(2), t2 = t2 === "Once" ? t2 : t2.replace(/Once$/, ""), ue(e, t2[0].toLowerCase() + t2.slice(1)) || ue(e, H(t2)) || ue(e, t2));
  }
  function Gt2(e) {
    let { type: t2, vnode: n2, proxy: r2, withProxy: o, propsOptions: [s], slots: i, attrs: l2, emit: a, render: h2, renderCache: f2, props: u, data: E3, setupState: y4, ctx: v4, inheritAttrs: N2 } = e, M4 = Pt2(e), U2, V3;
    try {
      if (n2.shapeFlag & 4) {
        let _ = o || r2, $3 = _;
        U2 = ve3(h2.call($3, _, f2, u, y4, E3, v4)), V3 = l2;
      } else {
        let _ = t2;
        U2 = ve3(_.length > 1 ? _(u, { attrs: l2, slots: i, emit: a }) : _(u, null)), V3 = t2.props ? l2 : Ni(l2);
      }
    } catch (_) {
      Le2.length = 0, yt2(_, e, 1), U2 = se3(oe3);
    }
    let d2 = U2, g4;
    if (V3 && N2 !== false) {
      let _ = Object.keys(V3), { shapeFlag: $3 } = d2;
      _.length && $3 & 7 && (s && _.some(fe) && (V3 = vi(V3, s)), d2 = We3(d2, V3, false, true));
    }
    if (n2.dirs && (d2 = We3(d2, null, false, true), d2.dirs = d2.dirs ? d2.dirs.concat(n2.dirs) : n2.dirs), n2.transition) {
      let _ = _n(d2.type) && zt2(d2) || d2;
      gt2(_, n2.transition);
    }
    return U2 = d2, Pt2(M4), U2;
  }
  var Ni = (e) => {
    let t2;
    for (let n2 in e) (n2 === "class" || n2 === "style" || pe(n2)) && ((t2 || (t2 = {}))[n2] = e[n2]);
    return t2;
  };
  var vi = (e, t2) => {
    let n2 = {};
    for (let r2 in e) (!fe(r2) || !(r2.slice(9) in t2)) && (n2[r2] = e[r2]);
    return n2;
  };
  function Oi(e, t2, n2) {
    let { props: r2, children: o, component: s } = e, { props: i, children: l2, patchFlag: a } = t2, h2 = s.emitsOptions;
    if (t2.dirs || t2.transition) return true;
    if (n2 && a >= 0) {
      if (a & 1024) return true;
      if (a & 16) return r2 ? kr(r2, i, h2) : !!i;
      if (a & 8) {
        let f2 = t2.dynamicProps;
        for (let u = 0; u < f2.length; u++) {
          let E3 = f2[u];
          if (fo(i, r2, E3) && !mn(h2, E3)) return true;
        }
      }
    } else return (o || l2) && (!l2 || !l2.$stable) ? true : r2 === i ? false : r2 ? i ? kr(r2, i, h2) : true : !!i;
    return false;
  }
  function kr(e, t2, n2) {
    let r2 = Object.keys(t2);
    if (r2.length !== Object.keys(e).length) return true;
    for (let o = 0; o < r2.length; o++) {
      let s = r2[o];
      if (fo(t2, e, s) && !mn(n2, s)) return true;
    }
    return false;
  }
  function fo(e, t2, n2) {
    let r2 = e[n2], o = t2[n2];
    return n2 === "style" && f(r2) && f(o) ? !b(r2, o) : r2 !== o;
  }
  function En({ vnode: e, parent: t2, suspense: n2 }, r2) {
    for (; t2; ) {
      let o = t2.subTree;
      if (o.suspense && o.suspense.activeBranch === e && (o.suspense.vnode.el = o.el = r2, e = o), o === e) (e = t2.vnode).el = r2, t2 = t2.parent;
      else break;
    }
    n2 && n2.activeBranch === e && (n2.vnode.el = r2);
  }
  var po = {};
  var ho = () => Object.create(po);
  var _o = (e) => Object.getPrototypeOf(e) === po;
  function bi(e, t2, n2, r2 = false) {
    let o = {}, s = ho();
    e.propsDefaults = /* @__PURE__ */ Object.create(null), go(e, t2, o, s);
    for (let i in e.propsOptions[0]) i in o || (o[i] = void 0);
    n2 ? e.props = r2 ? o : Kt(o) : e.type.props ? e.props = o : e.props = s, e.attrs = s;
  }
  function Di(e, t2, n2, r2) {
    let { props: o, attrs: s, vnode: { patchFlag: i } } = e, l2 = p2(o), [a] = e.propsOptions, h2 = false;
    if ((r2 || i > 0) && !(i & 16)) {
      if (i & 8) {
        let f2 = e.vnode.dynamicProps;
        for (let u = 0; u < f2.length; u++) {
          let E3 = f2[u];
          if (mn(e.emitsOptions, E3)) continue;
          let y4 = t2[E3];
          if (a) if (ue(s, E3)) y4 !== s[E3] && (s[E3] = y4, h2 = true);
          else {
            let v4 = Ae(E3);
            o[v4] = Un(a, l2, v4, y4, e, false);
          }
          else y4 !== s[E3] && (s[E3] = y4, h2 = true);
        }
      }
    } else {
      go(e, t2, o, s) && (h2 = true);
      let f2;
      for (let u in l2) (!t2 || !ue(t2, u) && ((f2 = H(u)) === u || !ue(t2, f2))) && (a ? n2 && (n2[u] !== void 0 || n2[f2] !== void 0) && (o[u] = Un(a, l2, u, void 0, e, true)) : delete o[u]);
      if (s !== l2) for (let u in s) (!t2 || !ue(t2, u)) && (delete s[u], h2 = true);
    }
    h2 && x(e.attrs, "set", "");
  }
  function go(e, t2, n2, r2) {
    let [o, s] = e.propsOptions, i = false, l2;
    if (t2) for (let a in t2) {
      if (be(a)) continue;
      let h2 = t2[a], f2;
      o && ue(o, f2 = Ae(a)) ? !s || !s.includes(f2) ? n2[f2] = h2 : (l2 || (l2 = {}))[f2] = h2 : mn(e.emitsOptions, a) || (!(a in r2) || h2 !== r2[a]) && (r2[a] = h2, i = true);
    }
    if (s) {
      let a = p2(n2), h2 = l2 || se;
      for (let f2 = 0; f2 < s.length; f2++) {
        let u = s[f2];
        n2[u] = Un(o, a, u, h2[u], e, !ue(h2, u));
      }
    }
    return i;
  }
  function Un(e, t2, n2, r2, o, s) {
    let i = e[n2];
    if (i != null) {
      let l2 = ue(i, "default");
      if (l2 && r2 === void 0) {
        let a = i.default;
        if (i.type !== Function && !i.skipFactory && g(a)) {
          let { propsDefaults: h2 } = o;
          if (n2 in h2) r2 = h2[n2];
          else {
            let f2 = Nt2(o);
            r2 = h2[n2] = a.call(null, t2), f2();
          }
        } else r2 = a;
        o.ce && o.ce._setProp(n2, r2);
      }
      i[0] && (s && !l2 ? r2 = false : i[1] && (r2 === "" || r2 === H(n2)) && (r2 = true));
    }
    return r2;
  }
  var Vi = /* @__PURE__ */ new WeakMap();
  function mo(e, t2, n2 = false) {
    let r2 = n2 ? Vi : t2.propsCache, o = r2.get(e);
    if (o) return o;
    let s = e.props, i = {}, l2 = [], a = false;
    if (!g(e)) {
      let f2 = (u) => {
        a = true;
        let [E3, y4] = mo(u, t2, true);
        de(i, E3), y4 && l2.push(...y4);
      };
      !n2 && t2.mixins.length && t2.mixins.forEach(f2), e.extends && f2(e.extends), e.mixins && e.mixins.forEach(f2);
    }
    if (!s && !a) return f(e) && r2.set(e, ae), ae;
    if (d(s)) for (let f2 = 0; f2 < s.length; f2++) {
      let u = Ae(s[f2]);
      Ar(u) && (i[u] = se);
    }
    else if (s) for (let f2 in s) {
      let u = Ae(f2);
      if (Ar(u)) {
        let E3 = s[f2], y4 = i[u] = d(E3) || g(E3) ? { type: E3 } : de({}, E3), v4 = y4.type, N2 = false, M4 = true;
        if (d(v4)) for (let U2 = 0; U2 < v4.length; ++U2) {
          let V3 = v4[U2], d2 = g(V3) && V3.name;
          if (d2 === "Boolean") {
            N2 = true;
            break;
          } else d2 === "String" && (M4 = false);
        }
        else N2 = g(v4) && v4.name === "Boolean";
        y4[0] = N2, y4[1] = M4, (N2 || ue(y4, "default")) && l2.push(u);
      }
    }
    let h2 = [i, l2];
    return f(e) && r2.set(e, h2), h2;
  }
  function Ar(e) {
    return e[0] !== "$" && !be(e);
  }
  var cr = (e) => e === "_" || e === "_ctx" || e === "$stable";
  var ur = (e) => d(e) ? e.map(ve3) : [ve3(e)];
  var wi = (e, t2, n2) => {
    if (t2._n) return t2;
    let r2 = Jr((...o) => ur(t2(...o)), n2);
    return r2._c = false, r2;
  };
  var Eo = (e, t2, n2) => {
    let r2 = e._ctx;
    for (let o in e) {
      if (cr(o)) continue;
      let s = e[o];
      if (g(s)) t2[o] = wi(o, s, r2);
      else if (s != null) {
        let i = ur(s);
        t2[o] = () => i;
      }
    }
  };
  var yo = (e, t2) => {
    let n2 = ur(t2);
    e.slots.default = () => n2;
  };
  var No = (e, t2, n2) => {
    for (let r2 in t2) (n2 || !cr(r2)) && (e[r2] = t2[r2]);
  };
  var xi = (e, t2, n2) => {
    let r2 = e.slots = ho();
    if (e.vnode.shapeFlag & 32) {
      let o = t2._;
      o ? (No(r2, t2, n2), n2 && xe(r2, "_", o, true)) : Eo(t2, r2);
    } else t2 && yo(e, t2);
  };
  var Ti = (e, t2, n2) => {
    let { vnode: r2, slots: o } = e, s = true, i = se;
    if (r2.shapeFlag & 32) {
      let l2 = t2._;
      l2 ? n2 && l2 === 1 ? s = false : No(o, t2, n2) : (s = !t2.$stable, Eo(t2, o)), i = t2;
    } else t2 && (yo(e, t2), i = { default: 1 });
    if (s) for (let l2 in o) !cr(l2) && i[l2] == null && delete o[l2];
  };
  function Ci() {
    let e = [];
    false, false, false;
  }
  var ce3 = wo;
  function Kl(e) {
    return vo(e);
  }
  function vo(e, t2) {
    Ci();
    let n2 = _e();
    n2.__VUE__ = true, false;
    let { insert: r2, remove: o, patchProp: s, createElement: i, createText: l2, createComment: a, setText: h2, setElementText: f2, parentNode: u, nextSibling: E3, setScopeId: y4 = ce, insertStaticContent: v4 } = e, N2 = (c, p3, m3, x2 = null, O2 = null, b4 = null, A3 = void 0, k3 = null, T2 = !!p3.dynamicChildren) => {
      if (c === p3) return;
      c && !Ce3(c, p3) && (x2 = Ut2(c), ke4(c, O2, b4, true), c = null), p3.patchFlag === -2 && (T2 = false, p3.dynamicChildren = null);
      let { type: D3, ref: L3, shapeFlag: R3 } = p3;
      switch (D3) {
        case Ge3:
          M4(c, p3, m3, x2);
          break;
        case oe3:
          U2(c, p3, m3, x2);
          break;
        case it:
          c == null && V3(p3, m3, x2, A3);
          break;
        case ge3:
          J4(c, p3, m3, x2, O2, b4, A3, k3, T2);
          break;
        default:
          R3 & 1 ? $3(c, p3, m3, x2, O2, b4, A3, k3, T2) : R3 & 6 ? G(c, p3, m3, x2, O2, b4, A3, k3, T2) : (R3 & 64 || R3 & 128) && D3.process(c, p3, m3, x2, O2, b4, A3, k3, T2, ut);
      }
      L3 != null && O2 ? dt2(L3, c && c.ref, b4, p3 || c, !p3) : L3 == null && c && c.ref != null && dt2(c.ref, null, b4, c, true);
    }, M4 = (c, p3, m3, x2) => {
      if (c == null) r2(p3.el = l2(p3.children), m3, x2);
      else {
        let O2 = p3.el = c.el;
        p3.children !== c.children && h2(O2, p3.children);
      }
    }, U2 = (c, p3, m3, x2) => {
      c == null ? r2(p3.el = a(p3.children || ""), m3, x2) : p3.el = c.el;
    }, V3 = (c, p3, m3, x2) => {
      [c.el, c.anchor] = v4(c.children, p3, m3, x2, c.el, c.anchor);
    }, d2 = (c, p3, m3, x2) => {
      if (p3.children !== c.children) {
        let O2 = E3(c.anchor);
        _(c), [p3.el, p3.anchor] = v4(p3.children, m3, O2, x2);
      } else p3.el = c.el, p3.anchor = c.anchor;
    }, g4 = ({ el: c, anchor: p3 }, m3, x2) => {
      let O2;
      for (; c && c !== p3; ) O2 = E3(c), r2(c, m3, x2), c = O2;
      r2(p3, m3, x2);
    }, _ = ({ el: c, anchor: p3 }) => {
      let m3;
      for (; c && c !== p3; ) m3 = E3(c), o(c), c = m3;
      o(p3);
    }, $3 = (c, p3, m3, x2, O2, b4, A3, k3, T2) => {
      if (p3.type === "svg" ? A3 = "svg" : p3.type === "math" && (A3 = "mathml"), c == null) C2(p3, m3, x2, O2, b4, A3, k3, T2);
      else {
        let D3 = c.el && c.el._isVueCE ? c.el : null;
        try {
          D3 && D3._beginPatch(), S3(c, p3, O2, b4, A3, k3, T2);
        } finally {
          D3 && D3._endPatch();
        }
      }
    }, C2 = (c, p3, m3, x2, O2, b4, A3, k3) => {
      let T2, D3, { props: L3, shapeFlag: R3, transition: F4, dirs: j3 } = c;
      if (T2 = c.el = i(c.type, b4, L3 && L3.is, L3), R3 & 8 ? f2(T2, c.children) : R3 & 16 && w4(c.children, T2, null, x2, O2, Tn(c, b4), A3, k3), j3 && Re(c, null, x2, "created"), I2(T2, c, c.scopeId, A3, x2), L3) {
        for (let z2 in L3) z2 !== "value" && !be(z2) && s(T2, z2, null, L3[z2], b4, x2);
        "value" in L3 && s(T2, "value", null, L3.value, b4), (D3 = L3.onVnodeBeforeMount) && Ne3(D3, x2, c);
      }
      false, j3 && Re(c, null, x2, "beforeMount");
      let Q3 = Oo(O2, F4);
      Q3 && F4.beforeEnter(T2), r2(T2, p3, m3), ((D3 = L3 && L3.onVnodeMounted) || Q3 || j3) && ce3(() => {
        let Z3;
        D3 && Ne3(D3, x2, c), Q3 && F4.enter(T2), j3 && Re(c, null, x2, "mounted");
      }, O2);
    }, I2 = (c, p3, m3, x2, O2) => {
      if (m3 && y4(c, m3), x2) for (let b4 = 0; b4 < x2.length; b4++) y4(c, x2[b4]);
      if (O2) {
        let b4 = O2.subTree;
        if (p3 === b4 || on(b4.type) && (b4.ssContent === p3 || b4.ssFallback === p3)) {
          let A3 = O2.vnode;
          I2(c, A3, A3.scopeId, A3.slotScopeIds, O2.parent);
        }
      }
    }, w4 = (c, p3, m3, x2, O2, b4, A3, k3, T2 = 0) => {
      for (let D3 = T2; D3 < c.length; D3++) {
        let L3 = c[D3] = k3 ? Fe2(c[D3]) : ve3(c[D3]);
        N2(null, L3, p3, m3, x2, O2, b4, A3, k3);
      }
    }, S3 = (c, p3, m3, x2, O2, b4, A3) => {
      let k3 = p3.el = c.el;
      let { patchFlag: T2, dynamicChildren: D3, dirs: L3 } = p3;
      T2 |= c.patchFlag & 16;
      let R3 = c.props || se, F4 = p3.props || se, j3;
      if (m3 && ze3(m3, false), (j3 = F4.onVnodeBeforeUpdate) && Ne3(j3, m3, p3, c), L3 && Re(p3, c, m3, "beforeUpdate"), m3 && ze3(m3, true), D3 && (!c.dynamicChildren || c.dynamicChildren.length !== D3.length) && (T2 = 0, A3 = false, D3 = null), (R3.innerHTML && F4.innerHTML == null || R3.textContent && F4.textContent == null) && f2(k3, ""), D3 ? P2(c.dynamicChildren, D3, k3, m3, x2, Tn(p3, O2), b4) : A3 || re2(c, p3, k3, null, m3, x2, Tn(p3, O2), b4, false), T2 > 0) {
        if (T2 & 16) W4(k3, R3, F4, m3, O2);
        else if (T2 & 2 && R3.class !== F4.class && s(k3, "class", null, F4.class, O2), T2 & 4 && s(k3, "style", R3.style, F4.style, O2), T2 & 8) {
          let Q3 = p3.dynamicProps;
          for (let z2 = 0; z2 < Q3.length; z2++) {
            let Z3 = Q3[z2], le2 = R3[Z3], ae4 = F4[Z3];
            (ae4 !== le2 || Z3 === "value") && s(k3, Z3, le2, ae4, O2, m3);
          }
        }
        T2 & 1 && c.children !== p3.children && f2(k3, p3.children);
      } else !A3 && D3 == null && W4(k3, R3, F4, m3, O2);
      ((j3 = F4.onVnodeUpdated) || L3) && ce3(() => {
        j3 && Ne3(j3, m3, p3, c), L3 && Re(p3, c, m3, "updated");
      }, x2);
    }, P2 = (c, p3, m3, x2, O2, b4, A3) => {
      for (let k3 = 0; k3 < p3.length; k3++) {
        let T2 = c[k3], D3 = p3[k3], L3 = T2.el && (T2.type === ge3 || !Ce3(T2, D3) || T2.shapeFlag & 198) ? u(T2.el) : m3;
        N2(T2, D3, L3, null, x2, O2, b4, A3, true);
      }
    }, W4 = (c, p3, m3, x2, O2) => {
      if (p3 !== m3) {
        if (p3 !== se) for (let b4 in p3) !be(b4) && !(b4 in m3) && s(c, b4, p3[b4], null, O2, x2);
        for (let b4 in m3) {
          if (be(b4)) continue;
          let A3 = m3[b4], k3 = p3[b4];
          A3 !== k3 && b4 !== "value" && s(c, b4, k3, A3, O2, x2);
        }
        "value" in m3 && s(c, "value", p3.value, m3.value, O2);
      }
    }, J4 = (c, p3, m3, x2, O2, b4, A3, k3, T2) => {
      let D3 = p3.el = c ? c.el : l2(""), L3 = p3.anchor = c ? c.anchor : l2(""), { patchFlag: R3, dynamicChildren: F4, slotScopeIds: j3 } = p3;
      j3 && (k3 = k3 ? k3.concat(j3) : j3), c == null ? (r2(D3, m3, x2), r2(L3, m3, x2), w4(p3.children || [], m3, L3, O2, b4, A3, k3, T2)) : R3 > 0 && R3 & 64 && F4 && c.dynamicChildren && c.dynamicChildren.length === F4.length ? (P2(c.dynamicChildren, F4, m3, O2, b4, A3, k3), (p3.key != null || O2 && p3 === O2.subTree) && ar(c, p3, true)) : re2(c, p3, m3, L3, O2, b4, A3, k3, T2);
    }, G = (c, p3, m3, x2, O2, b4, A3, k3, T2) => {
      p3.slotScopeIds = k3, c == null ? p3.shapeFlag & 512 ? O2.ctx.activate(p3, m3, x2, A3, T2) : ee(p3, m3, x2, O2, b4, A3, T2) : H2(c, p3, T2);
    }, ee = (c, p3, m3, x2, O2, b4, A3) => {
      let k3 = c.component = ko(c, x2, O2);
      if (Ft2(c) && (k3.ctx.renderer = ut), Po(k3, false, A3), k3.asyncDep) {
        if (O2 && O2.registerDep(k3, Y4, A3), !c.el) {
          let T2 = k3.subTree = se3(oe3);
          U2(null, T2, p3, m3), c.placeholder = T2.el;
        }
      } else Y4(k3, c, p3, m3, O2, b4, A3);
    }, H2 = (c, p3, m3) => {
      let x2 = p3.component = c.component;
      if (Oi(c, p3, m3)) if (x2.asyncDep && !x2.asyncResolved) {
        B3(x2, p3, m3);
        return;
      } else x2.next = p3, x2.update();
      else p3.el = c.el, x2.vnode = p3;
    }, Y4 = (c, p3, m3, x2, O2, b4, A3) => {
      let k3 = () => {
        if (c.isMounted) {
          let { next: R3, bu: F4, u: j3, parent: Q3, vnode: z2 } = c;
          {
            let Oe3 = bo(c);
            if (Oe3) {
              R3 && (R3.el = z2.el, B3(c, R3, A3)), Oe3.asyncDep.then(() => {
                ce3(() => {
                  c.isUnmounted || D3();
                }, O2);
              });
              return;
            }
          }
          let Z3 = R3, le2;
          ze3(c, false), R3 ? (R3.el = z2.el, B3(c, R3, A3)) : R3 = z2, F4 && Oe(F4), (le2 = R3.props && R3.props.onVnodeBeforeUpdate) && Ne3(le2, Q3, R3, z2), ze3(c, true);
          let ae4 = Gt2(c), xe2 = c.subTree;
          c.subTree = ae4, N2(xe2, ae4, u(xe2.el), Ut2(xe2), c, O2, b4), R3.el = ae4.el, Z3 === null && En(c, ae4.el), j3 && ce3(j3, O2), (le2 = R3.props && R3.props.onVnodeUpdated) && ce3(() => Ne3(le2, Q3, R3, z2), O2), false;
        } else {
          let R3, { el: F4, props: j3 } = p3, { bm: Q3, m: z2, parent: Z3, root: le2, type: ae4 } = c, xe2 = Ue2(p3);
          if (ze3(c, false), Q3 && Oe(Q3), !xe2 && (R3 = j3 && j3.onVnodeBeforeMount) && Ne3(R3, Z3, p3), ze3(c, true), F4 && On) {
            let Oe3 = () => {
              c.subTree = Gt2(c), On(F4, c.subTree, c, O2, null);
            };
            xe2 && ae4.__asyncHydrate ? ae4.__asyncHydrate(F4, c, Oe3) : Oe3();
          } else {
            le2.ce && le2.ce._hasShadowRoot() && le2.ce._injectChildStyle(ae4, c.parent ? c.parent.type : void 0);
            let Oe3 = c.subTree = Gt2(c);
            N2(null, Oe3, m3, x2, c, O2, b4), p3.el = Oe3.el;
          }
          if (z2 && ce3(z2, O2), !xe2 && (R3 = j3 && j3.onVnodeMounted)) {
            let Oe3 = p3;
            ce3(() => Ne3(R3, Z3, Oe3), O2);
          }
          (p3.shapeFlag & 256 || Z3 && Ue2(Z3.vnode) && Z3.vnode.shapeFlag & 256) && c.a && ce3(c.a, O2), c.isMounted = true, false, p3 = m3 = x2 = null;
        }
      };
      c.scope.on();
      let T2 = c.effect = new k2(k3);
      c.scope.off();
      let D3 = c.update = T2.run.bind(T2), L3 = c.job = T2.runIfDirty.bind(T2);
      L3.i = c, L3.id = c.uid, T2.scheduler = () => Xn(L3), ze3(c, true), D3();
    }, B3 = (c, p3, m3) => {
      p3.component = c;
      let x2 = c.vnode.props;
      c.vnode = p3, c.next = null, Di(c, p3.props, x2, m3), Ti(c, p3.children, m3), Ue(), Er(c), $e();
    }, re2 = (c, p3, m3, x2, O2, b4, A3, k3, T2 = false) => {
      let D3 = c && c.children, L3 = c ? c.shapeFlag : 0, R3 = p3.children, { patchFlag: F4, shapeFlag: j3 } = p3;
      if (F4 > 0) {
        if (F4 & 128) {
          _e3(D3, R3, m3, x2, O2, b4, A3, k3, T2);
          return;
        } else if (F4 & 256) {
          te2(D3, R3, m3, x2, O2, b4, A3, k3, T2);
          return;
        }
      }
      j3 & 8 ? (L3 & 16 && vt3(D3, O2, b4), R3 !== D3 && f2(m3, R3)) : L3 & 16 ? j3 & 16 ? _e3(D3, R3, m3, x2, O2, b4, A3, k3, T2) : vt3(D3, O2, b4, true) : (L3 & 8 && f2(m3, ""), j3 & 16 && w4(R3, m3, x2, O2, b4, A3, k3, T2));
    }, te2 = (c, p3, m3, x2, O2, b4, A3, k3, T2) => {
      c = c || ae, p3 = p3 || ae;
      let D3 = c.length, L3 = p3.length, R3 = Math.min(D3, L3), F4;
      for (F4 = 0; F4 < R3; F4++) {
        let j3 = p3[F4] = T2 ? Fe2(p3[F4]) : ve3(p3[F4]);
        N2(c[F4], j3, m3, null, O2, b4, A3, k3, T2);
      }
      D3 > L3 ? vt3(c, O2, b4, true, false, R3) : w4(p3, m3, x2, O2, b4, A3, k3, T2, R3);
    }, _e3 = (c, p3, m3, x2, O2, b4, A3, k3, T2) => {
      let D3 = 0, L3 = p3.length, R3 = c.length - 1, F4 = L3 - 1;
      for (; D3 <= R3 && D3 <= F4; ) {
        let j3 = c[D3], Q3 = p3[D3] = T2 ? Fe2(p3[D3]) : ve3(p3[D3]);
        if (Ce3(j3, Q3)) N2(j3, Q3, m3, null, O2, b4, A3, k3, T2);
        else break;
        D3++;
      }
      for (; D3 <= R3 && D3 <= F4; ) {
        let j3 = c[R3], Q3 = p3[F4] = T2 ? Fe2(p3[F4]) : ve3(p3[F4]);
        if (Ce3(j3, Q3)) N2(j3, Q3, m3, null, O2, b4, A3, k3, T2);
        else break;
        R3--, F4--;
      }
      if (D3 > R3) {
        if (D3 <= F4) {
          let j3 = F4 + 1, Q3 = j3 < L3 ? p3[j3].el : x2;
          for (; D3 <= F4; ) N2(null, p3[D3] = T2 ? Fe2(p3[D3]) : ve3(p3[D3]), m3, Q3, O2, b4, A3, k3, T2), D3++;
        }
      } else if (D3 > F4) for (; D3 <= R3; ) ke4(c[D3], O2, b4, true), D3++;
      else {
        let j3 = D3, Q3 = D3, z2 = /* @__PURE__ */ new Map();
        for (D3 = Q3; D3 <= F4; D3++) {
          let be3 = p3[D3] = T2 ? Fe2(p3[D3]) : ve3(p3[D3]);
          be3.key != null && z2.set(be3.key, D3);
        }
        let Z3, le2 = 0, ae4 = F4 - Q3 + 1, xe2 = false, Oe3 = 0, Ot3 = new Array(ae4);
        for (D3 = 0; D3 < ae4; D3++) Ot3[D3] = 0;
        for (D3 = j3; D3 <= R3; D3++) {
          let be3 = c[D3];
          if (le2 >= ae4) {
            ke4(be3, O2, b4, true);
            continue;
          }
          let Ae3;
          if (be3.key != null) Ae3 = z2.get(be3.key);
          else for (Z3 = Q3; Z3 <= F4; Z3++) if (Ot3[Z3 - Q3] === 0 && Ce3(be3, p3[Z3])) {
            Ae3 = Z3;
            break;
          }
          Ae3 === void 0 ? ke4(be3, O2, b4, true) : (Ot3[Ae3 - Q3] = D3 + 1, Ae3 >= Oe3 ? Oe3 = Ae3 : xe2 = true, N2(be3, p3[Ae3], m3, null, O2, b4, A3, k3, T2), le2++);
        }
        let dr = xe2 ? $i(Ot3) : ae;
        for (Z3 = dr.length - 1, D3 = ae4 - 1; D3 >= 0; D3--) {
          let be3 = Q3 + D3, Ae3 = p3[be3], hr = p3[be3 + 1], _r = be3 + 1 < L3 ? hr.el || Do(hr) : x2;
          Ot3[D3] === 0 ? N2(null, Ae3, m3, _r, O2, b4, A3, k3, T2) : xe2 && (Z3 < 0 || D3 !== dr[Z3] ? ie4(Ae3, m3, _r, 2) : Z3--);
        }
      }
    }, ie4 = (c, p3, m3, x2, O2 = null) => {
      let { el: b4, type: A3, transition: k3, children: T2, shapeFlag: D3 } = c;
      if (D3 & 6) {
        ie4(c.component.subTree, p3, m3, x2);
        return;
      }
      if (D3 & 128) {
        c.suspense.move(p3, m3, x2);
        return;
      }
      if (D3 & 64) {
        A3.move(c, p3, m3, ut);
        return;
      }
      if (A3 === ge3) {
        r2(b4, p3, m3);
        for (let R3 = 0; R3 < T2.length; R3++) ie4(T2[R3], p3, m3, x2);
        r2(c.anchor, p3, m3);
        return;
      }
      if (A3 === it) {
        g4(c, p3, m3);
        return;
      }
      if (x2 !== 2 && D3 & 1 && k3) if (x2 === 0) k3.persisted && !b4[Ve3] ? r2(b4, p3, m3) : (k3.beforeEnter(b4), r2(b4, p3, m3), ce3(() => k3.enter(b4), O2));
      else {
        let { leave: R3, delayLeave: F4, afterLeave: j3 } = k3, Q3 = () => {
          c.ctx.isUnmounted ? o(b4) : r2(b4, p3, m3);
        }, z2 = () => {
          let Z3 = b4._isLeaving || !!b4[Ve3];
          b4._isLeaving && b4[Ve3](true), k3.persisted && !Z3 ? Q3() : R3(b4, () => {
            Q3(), j3 && j3();
          });
        };
        F4 ? F4(b4, Q3, z2) : z2();
      }
      else r2(b4, p3, m3);
    }, ke4 = (c, p3, m3, x2 = false, O2 = false) => {
      let { type: b4, props: A3, ref: k3, children: T2, dynamicChildren: D3, shapeFlag: L3, patchFlag: R3, dirs: F4, cacheIndex: j3, memo: Q3 } = c;
      if (R3 === -2 && (O2 = false), k3 != null && (Ue(), dt2(k3, null, m3, c, true), $e()), j3 != null && (p3.renderCache[j3] = void 0), L3 & 256) {
        p3.ctx.deactivate(c);
        return;
      }
      let z2 = L3 & 1 && F4, Z3 = !Ue2(c), le2;
      if (Z3 && (le2 = A3 && A3.onVnodeBeforeUnmount) && Ne3(le2, p3, c), L3 & 6) Fo(c.component, m3, x2);
      else {
        if (L3 & 128) {
          c.suspense.unmount(m3, x2);
          return;
        }
        z2 && Re(c, null, p3, "beforeUnmount"), L3 & 64 ? c.type.remove(c, p3, m3, ut, x2) : D3 && !D3.hasOnce && (b4 !== ge3 || R3 > 0 && R3 & 64) ? vt3(D3, p3, m3, false, true) : (b4 === ge3 && R3 & 384 || !O2 && L3 & 16) && vt3(T2, p3, m3), x2 && fr(c);
      }
      let ae4 = Q3 != null && j3 == null;
      (Z3 && (le2 = A3 && A3.onVnodeUnmounted) || z2 || ae4) && ce3(() => {
        le2 && Ne3(le2, p3, c), z2 && Re(c, null, p3, "unmounted"), ae4 && (c.el = null);
      }, m3);
    }, fr = (c) => {
      let { type: p3, el: m3, anchor: x2, transition: O2 } = c;
      if (p3 === ge3) {
        Mo(m3, x2);
        return;
      }
      if (p3 === it) {
        _(c);
        return;
      }
      let b4 = () => {
        o(m3), O2 && !O2.persisted && O2.afterLeave && O2.afterLeave();
      };
      if (c.shapeFlag & 1 && O2 && !O2.persisted) {
        let { leave: A3, delayLeave: k3 } = O2, T2 = () => A3(m3, b4);
        k3 ? k3(c.el, b4, T2) : T2();
      } else b4();
    }, Mo = (c, p3) => {
      let m3;
      for (; c !== p3; ) m3 = E3(c), o(c), c = m3;
      o(p3);
    }, Fo = (c, p3, m3) => {
      let { bum: x2, scope: O2, job: b4, subTree: A3, um: k3, m: T2, a: D3 } = c;
      rn(T2), rn(D3), x2 && Oe(x2), O2.stop(), b4 && (b4.flags |= 8, ke4(A3, c, p3, m3)), k3 && ce3(k3, p3), ce3(() => {
        c.isUnmounted = true;
      }, p3), false;
    }, vt3 = (c, p3, m3, x2 = false, O2 = false, b4 = 0) => {
      for (let A3 = b4; A3 < c.length; A3++) ke4(c[A3], p3, m3, x2, O2);
    }, Ut2 = (c) => {
      if (c.shapeFlag & 6) return Ut2(c.component.subTree);
      if (c.shapeFlag & 128) return c.suspense.next();
      let p3 = E3(c.anchor || c.el), m3 = p3 && p3[Xr];
      return m3 ? E3(m3) : p3;
    }, Nn = false, pr = (c, p3, m3) => {
      let x2;
      c == null ? p3._vnode && (ke4(p3._vnode, null, null, true), x2 = p3._vnode.component) : N2(p3._vnode || null, c, p3, null, null, null, m3), p3._vnode = c, Nn || (Nn = true, Er(x2), Zt(), Nn = false);
    }, ut = { p: N2, um: ke4, m: ie4, r: fr, mt: ee, mc: w4, pc: re2, pbc: P2, n: Ut2, o: e }, vn2, On;
    return t2 && ([vn2, On] = t2(ut)), { render: pr, hydrate: vn2, createApp: gi(pr, vn2) };
  }
  function Tn({ type: e, props: t2 }, n2) {
    return n2 === "svg" && e === "foreignObject" || n2 === "mathml" && e === "annotation-xml" && t2 && t2.encoding && t2.encoding.includes("html") ? void 0 : n2;
  }
  function ze3({ effect: e, job: t2 }, n2) {
    n2 ? (e.flags |= 32, t2.flags |= 4) : (e.flags &= -33, t2.flags &= -5);
  }
  function Oo(e, t2) {
    return (!e || e && !e.pendingBranch) && t2 && !t2.persisted;
  }
  function ar(e, t2, n2 = false) {
    let r2 = e.children, o = t2.children;
    if (d(r2) && d(o)) for (let s = 0; s < r2.length; s++) {
      let i = r2[s], l2 = o[s];
      l2.shapeFlag & 1 && !l2.dynamicChildren && ((l2.patchFlag <= 0 || l2.patchFlag === 32) && (l2 = o[s] = Fe2(o[s]), l2.el = i.el), !n2 && l2.patchFlag !== -2 && ar(i, l2)), l2.type === Ge3 && (l2.patchFlag === -1 && (l2 = o[s] = Fe2(l2)), l2.el = i.el), l2.type === oe3 && !l2.el && (l2.el = i.el);
    }
  }
  function $i(e) {
    let t2 = e.slice(), n2 = [0], r2, o, s, i, l2, a = e.length;
    for (r2 = 0; r2 < a; r2++) {
      let h2 = e[r2];
      if (h2 !== 0) {
        if (o = n2[n2.length - 1], e[o] < h2) {
          t2[r2] = o, n2.push(r2);
          continue;
        }
        for (s = 0, i = n2.length - 1; s < i; ) l2 = s + i >> 1, e[n2[l2]] < h2 ? s = l2 + 1 : i = l2;
        h2 < e[n2[s]] && (s > 0 && (t2[r2] = n2[s - 1]), n2[s] = r2);
      }
    }
    for (s = n2.length, i = n2[s - 1]; s-- > 0; ) n2[s] = i, i = t2[i];
    return n2;
  }
  function bo(e) {
    let t2 = e.subTree.component;
    if (t2) return t2.asyncDep && !t2.asyncResolved ? t2 : bo(t2);
  }
  function rn(e) {
    if (e) for (let t2 = 0; t2 < e.length; t2++) e[t2].flags |= 8;
  }
  function Do(e) {
    if (e.placeholder) return e.placeholder;
    let t2 = e.component;
    return t2 ? Do(t2.subTree) : null;
  }
  var on = (e) => e.__isSuspense;
  function wo(e, t2) {
    t2 && t2.pendingBranch ? d(e) ? t2.effects.push(...e) : t2.effects.push(e) : $n(e);
  }
  var ge3 = /* @__PURE__ */ Symbol.for("v-fgt");
  var Ge3 = /* @__PURE__ */ Symbol.for("v-txt");
  var oe3 = /* @__PURE__ */ Symbol.for("v-cmt");
  var it = /* @__PURE__ */ Symbol.for("v-stc");
  var Le2 = [];
  var me3 = null;
  function sn(e = false) {
    Le2.push(me3 = e ? null : []);
  }
  function yn() {
    Le2.pop(), me3 = Le2[Le2.length - 1] || null;
  }
  var lt2 = 1;
  function ln(e, t2 = false) {
    lt2 += e, e < 0 && me3 && t2 && (me3.hasOnce = true);
  }
  function xo(e) {
    return e.dynamicChildren = lt2 > 0 ? me3 || ae : null, yn(), lt2 > 0 && me3 && me3.push(e), e;
  }
  function Jl(e, t2, n2, r2, o, s) {
    return xo(Co(e, t2, n2, r2, o, s, true));
  }
  function Bn(e, t2, n2, r2, o) {
    return xo(se3(e, t2, n2, r2, o, true));
  }
  function Qe2(e) {
    return e ? e.__v_isVNode === true : false;
  }
  function Ce3(e, t2) {
    return e.type === t2.type && e.key === t2.key;
  }
  var To = ({ key: e }) => e ?? null;
  var Xt = ({ ref: e, ref_key: t2, ref_for: n2 }) => (typeof e == "number" && (e = "" + e), e != null ? p(e) || g2(e) || g(e) ? { i: de3, r: e, k: t2, f: !!n2 } : e : null);
  function Co(e, t2 = null, n2 = null, r2 = 0, o = null, s = e === ge3 ? 0 : 1, i = false, l2 = false) {
    let a = { __v_isVNode: true, __v_skip: true, type: e, props: t2, key: t2 && To(t2), ref: t2 && Xt(t2), scopeId: hn, slotScopeIds: null, children: n2, component: null, suspense: null, ssContent: null, ssFallback: null, dirs: null, transition: null, el: null, anchor: null, target: null, targetStart: null, targetAnchor: null, staticCount: 0, shapeFlag: s, patchFlag: r2, dynamicProps: o, dynamicChildren: null, appContext: null, ctx: de3 };
    return l2 ? (cn(a, n2), s & 128 && e.normalize(a)) : n2 && (a.shapeFlag |= p(n2) ? 8 : 16), lt2 > 0 && !i && me3 && (a.patchFlag > 0 || s & 6) && a.patchFlag !== 32 && me3.push(a), a;
  }
  var se3 = Fi;
  function Fi(e, t2 = null, n2 = null, r2 = 0, o = null, s = false) {
    if ((!e || e === so) && (e = oe3), Qe2(e)) {
      let l2 = We3(e, t2, true);
      return n2 && cn(l2, n2), lt2 > 0 && !s && me3 && (l2.shapeFlag & 6 ? me3[me3.indexOf(e)] = l2 : me3.push(l2)), l2.patchFlag = -2, l2;
    }
    if (Ji(e) && (e = e.__vccOpts), t2) {
      t2 = Hi(t2);
      let { class: l2, style: a } = t2;
      l2 && !p(l2) && (t2.class = w(l2)), f(a) && (qe(a) && !d(a) && (a = de({}, a)), t2.style = M(a));
    }
    let i = p(e) ? 1 : on(e) ? 128 : _n(e) ? 64 : f(e) ? 4 : g(e) ? 2 : 0;
    return Co(e, t2, n2, r2, o, i, s, true);
  }
  function Hi(e) {
    return e ? qe(e) || _o(e) ? de({}, e) : e : null;
  }
  function We3(e, t2, n2 = false, r2 = false) {
    let { props: o, ref: s, patchFlag: i, children: l2, transition: a } = e, h2 = t2 ? Li(o || {}, t2) : o, f2 = { __v_isVNode: true, __v_skip: true, type: e.type, props: h2, key: h2 && To(h2), ref: t2 && t2.ref ? n2 && s ? d(s) ? s.concat(Xt(t2)) : [s, Xt(t2)] : Xt(t2) : s, scopeId: e.scopeId, slotScopeIds: e.slotScopeIds, children: l2, target: e.target, targetStart: e.targetStart, targetAnchor: e.targetAnchor, staticCount: e.staticCount, shapeFlag: e.shapeFlag, patchFlag: t2 && e.type !== ge3 ? i === -1 ? 16 : i | 16 : i, dynamicProps: e.dynamicProps, dynamicChildren: e.dynamicChildren, appContext: e.appContext, dirs: e.dirs, transition: a, component: e.component, suspense: e.suspense, ssContent: e.ssContent && We3(e.ssContent), ssFallback: e.ssFallback && We3(e.ssFallback), placeholder: e.placeholder, el: e.el, anchor: e.anchor, ctx: e.ctx, ce: e.ce };
    return a && r2 && gt2(f2, a.clone(f2)), f2;
  }
  function $o(e = " ", t2 = 0) {
    return se3(Ge3, null, e, t2);
  }
  function Ui(e = "", t2 = false) {
    return t2 ? (sn(), Bn(oe3, null, e)) : se3(oe3, null, e);
  }
  function ve3(e) {
    return e == null || typeof e == "boolean" ? se3(oe3) : d(e) ? se3(ge3, null, e.slice()) : Qe2(e) ? Fe2(e) : se3(Ge3, null, String(e));
  }
  function Fe2(e) {
    return e.el === null && e.patchFlag !== -1 || e.memo ? e : We3(e);
  }
  function cn(e, t2) {
    let n2 = 0, { shapeFlag: r2 } = e;
    if (t2 == null) t2 = null;
    else if (d(t2)) n2 = 16;
    else if (typeof t2 == "object") if (r2 & 65) {
      let o = t2.default;
      o && (o._c && (o._d = false), cn(e, o()), o._c && (o._d = true));
      return;
    } else {
      n2 = 32;
      let o = t2._;
      !o && !_o(t2) ? t2._ctx = de3 : o === 3 && de3 && (de3.slots._ === 1 ? t2._ = 1 : (t2._ = 2, e.patchFlag |= 1024));
    }
    else if (g(t2)) {
      if (r2 & 65) {
        cn(e, { default: t2 });
        return;
      }
      t2 = { default: t2, _ctx: de3 }, n2 = 32;
    } else t2 = String(t2), r2 & 64 ? (n2 = 16, t2 = [$o(t2)]) : n2 = 8;
    e.children = t2, e.shapeFlag |= n2;
  }
  function Li(...e) {
    let t2 = {};
    for (let n2 = 0; n2 < e.length; n2++) {
      let r2 = e[n2];
      for (let o in r2) if (o === "class") t2.class !== r2.class && (t2.class = w([t2.class, r2.class]));
      else if (o === "style") t2.style = M([t2.style, r2.style]);
      else if (pe(o)) {
        let s = t2[o], i = r2[o];
        i && s !== i && !(d(s) && s.includes(i)) ? t2[o] = s ? [].concat(s, i) : i : i == null && s == null && !fe(o) && (t2[o] = i);
      } else o !== "" && (t2[o] = r2[o]);
    }
    return t2;
  }
  function Ne3(e, t2, n2, r2 = null) {
    Ie2(e, t2, 7, [n2, r2]);
  }
  var Bi = co();
  var ji = 0;
  function ko(e, t2, n2) {
    let r2 = e.type, o = (t2 ? t2.appContext : e.appContext) || Bi, s = { uid: ji++, vnode: e, type: r2, parent: t2, appContext: o, root: null, next: null, subTree: null, effect: null, update: null, job: null, scope: new ge2(true), render: null, proxy: null, exposed: null, exposeProxy: null, withProxy: null, provides: t2 ? t2.provides : Object.create(o.provides), ids: t2 ? t2.ids : ["", 0, 0], accessCache: null, renderCache: [], components: null, directives: null, propsOptions: mo(r2, o), emitsOptions: ao(r2, o), emit: null, emitted: null, propsDefaults: se, inheritAttrs: r2.inheritAttrs, ctx: se, data: se, props: se, attrs: se, slots: se, refs: se, setupState: se, setupContext: null, suspense: n2, suspenseId: n2 ? n2.pendingId : 0, asyncDep: null, asyncResolved: false, isMounted: false, isUnmounted: false, isDeactivated: false, bc: null, c: null, bm: null, m: null, bu: null, u: null, um: null, bum: null, da: null, a: null, rtg: null, rtc: null, ec: null, sp: null };
    return s.ctx = { _: s }, s.root = t2 ? t2.root : s, s.emit = mi.bind(null, s), e.ce && e.ce(s), s;
  }
  var pe2 = null;
  var Me2 = () => pe2 || de3;
  var un;
  var Xe3;
  {
    let e = _e(), t2 = (n2, r2) => {
      let o;
      return (o = e[n2]) || (o = e[n2] = []), o.push(r2), (s) => {
        o.length > 1 ? o.forEach((i) => i(s)) : o[0](s);
      };
    };
    un = t2("__VUE_INSTANCE_SETTERS__", (n2) => pe2 = n2), Xe3 = t2("__VUE_SSR_SETTERS__", (n2) => ct2 = n2);
  }
  var Nt2 = (e) => {
    let t2 = pe2;
    return un(e), e.scope.on(), () => {
      e.scope.off(), un(t2);
    };
  };
  var It = () => {
    pe2 && pe2.scope.off(), un(null);
  };
  function Ao(e) {
    return e.vnode.shapeFlag & 4;
  }
  var ct2 = false;
  function Po(e, t2 = false, n2 = false) {
    t2 && Xe3(t2);
    let { props: r2, children: o } = e.vnode, s = Ao(e);
    bi(e, r2, s, t2), xi(e, o, n2 || t2);
    let i = s ? Wi(e, t2) : void 0;
    return t2 && Xe3(false), i;
  }
  function Wi(e, t2) {
    let n2 = e.type;
    e.accessCache = /* @__PURE__ */ Object.create(null), e.proxy = new Proxy(e.ctx, Mn);
    let { setup: r2 } = n2;
    if (r2) {
      Ue();
      let o = e.setupContext = r2.length > 1 ? Ro(e) : null, s = Nt2(e), i = Et2(r2, e, 0, [e.props, o]), l2 = ge(i);
      if ($e(), s(), (l2 || e.sp) && !Ue2(e) && zn(e), l2) {
        if (i.then(It, It), t2) return i.then((a) => {
          Xe3(true);
          try {
            jn(e, a, t2);
          } finally {
            Xe3(false);
          }
        }).catch((a) => {
          yt2(a, e, 0);
        });
        e.asyncDep = i;
      } else jn(e, i, t2);
    } else So(e, t2);
  }
  function jn(e, t2, n2) {
    g(t2) ? e.type.__ssrInlineRender ? e.ssrRender = t2 : e.render = t2 : f(t2) && (false, e.setupState = Yt(t2)), So(e, n2);
  }
  var an;
  var Wn;
  function So(e, t2, n2) {
    let r2 = e.type;
    if (!e.render) {
      if (!t2 && an && !r2.render) {
        let o = r2.template || lr(e).template;
        if (o) {
          let { isCustomElement: s, compilerOptions: i } = e.appContext.config, { delimiters: l2, compilerOptions: a } = r2, h2 = de(de({ isCustomElement: s, delimiters: l2 }, i), a);
          r2.render = an(o, h2);
        }
      }
      e.render = r2.render || ce, Wn && Wn(e);
    }
    if (true) {
      let o = Nt2(e);
      Ue();
      try {
        ai(e);
      } finally {
        $e(), o();
      }
    }
  }
  var Ki = { get(e, t2) {
    return w2(e, "get", ""), e[t2];
  } };
  function Ro(e) {
    let t2 = (n2) => {
      e.exposed = n2 || {};
    };
    return { attrs: new Proxy(e.attrs, Ki), slots: e.slots, emit: e.emit, expose: t2 };
  }
  function Ht2(e) {
    return e.exposed ? e.exposeProxy || (e.exposeProxy = new Proxy(Yt(jt(e.exposed)), { get(t2, n2) {
      if (n2 in t2) return t2[n2];
      if (n2 in $t2) return $t2[n2](e);
    }, has(t2, n2) {
      return n2 in t2 || n2 in $t2;
    } })) : e.proxy;
  }
  function Ji(e) {
    return g(e) && "__vccOpts" in e;
  }
  var Gi = (e, t2) => Jt(e, t2, ct2);
  function zl(e, t2, n2) {
    try {
      ln(-1);
      let r2 = arguments.length;
      return r2 === 2 ? f(t2) && !d(t2) ? Qe2(t2) ? se3(e, null, [t2]) : se3(e, t2) : se3(e, null, t2) : (r2 > 3 ? n2 = Array.prototype.slice.call(arguments, 2) : r2 === 3 && Qe2(n2) && (n2 = [n2]), se3(e, t2, n2));
    } finally {
      ln(1);
    }
  }
  var Sr = "3.5.42";

  // choysum-esm:https://esm.sh/@vue/runtime-dom@3.5.42/es2020/runtime-dom.mjs?target=es2020
  var et;
  var yt3 = typeof window < "u" && window.trustedTypes;
  if (yt3) try {
    et = yt3.createPolicy("vue", { createHTML: (t2) => t2 });
  } catch {
  }
  var Wt = et ? (t2) => et.createHTML(t2) : (t2) => t2;
  var Ke4 = "http://www.w3.org/2000/svg";
  var ze4 = "http://www.w3.org/1998/Math/MathML";
  var y3 = typeof document < "u" ? document : null;
  var bt3 = y3 && y3.createElement("template");
  var We4 = { insert: (t2, e, n2) => {
    e.insertBefore(t2, n2 || null);
  }, remove: (t2) => {
    let e = t2.parentNode;
    e && e.removeChild(t2);
  }, createElement: (t2, e, n2, s) => {
    let o = e === "svg" ? y3.createElementNS(Ke4, t2) : e === "mathml" ? y3.createElementNS(ze4, t2) : n2 ? y3.createElement(t2, { is: n2 }) : y3.createElement(t2);
    return t2 === "select" && s && s.multiple != null && o.setAttribute("multiple", s.multiple), o;
  }, createText: (t2) => y3.createTextNode(t2), createComment: (t2) => y3.createComment(t2), setText: (t2, e) => {
    t2.nodeValue = e;
  }, setElementText: (t2, e) => {
    t2.textContent = e;
  }, parentNode: (t2) => t2.parentNode, nextSibling: (t2) => t2.nextSibling, querySelector: (t2) => y3.querySelector(t2), setScopeId(t2, e) {
    t2.setAttribute(e, "");
  }, insertStaticContent(t2, e, n2, s, o, i) {
    let r2 = n2 ? n2.previousSibling : e.lastChild;
    if (o && (o === i || o.nextSibling)) for (; e.insertBefore(o.cloneNode(true), n2), !(o === i || !(o = o.nextSibling)); ) ;
    else {
      bt3.innerHTML = Wt(s === "svg" ? `<svg>${t2}</svg>` : s === "mathml" ? `<math>${t2}</math>` : t2);
      let c = bt3.content;
      if (s === "svg" || s === "mathml") {
        let a = c.firstChild;
        for (; a.firstChild; ) c.appendChild(a.firstChild);
        c.removeChild(a);
      }
      e.insertBefore(c, n2);
    }
    return [r2 ? r2.nextSibling : e.firstChild, n2 ? n2.previousSibling : e.lastChild];
  } };
  var S2 = "transition";
  var L2 = "animation";
  var M3 = /* @__PURE__ */ Symbol("_vtc");
  var kt = { name: String, type: String, css: { type: Boolean, default: true }, duration: [String, Number, Object], enterFromClass: String, enterActiveClass: String, enterToClass: String, appearFromClass: String, appearActiveClass: String, appearToClass: String, leaveFromClass: String, leaveActiveClass: String, leaveToClass: String };
  var Gt3 = de({}, Ss, kt);
  var ke3 = (t2) => (t2.displayName = "Transition", t2.props = Gt3, t2);
  var Un2 = ke3((t2, { slots: e }) => zl(_l, qt3(t2), e));
  var T = (t2, e = []) => {
    d(t2) ? t2.forEach((n2) => n2(...e)) : t2 && t2(...e);
  };
  var St3 = (t2) => t2 ? d(t2) ? t2.some((e) => e.length > 1) : t2.length > 1 : false;
  function qt3(t2) {
    let e = {};
    for (let u in t2) u in kt || (e[u] = t2[u]);
    if (t2.css === false) return e;
    let { name: n2 = "v", type: s, duration: o, enterFromClass: i = `${n2}-enter-from`, enterActiveClass: r2 = `${n2}-enter-active`, enterToClass: c = `${n2}-enter-to`, appearFromClass: a = i, appearActiveClass: l2 = r2, appearToClass: f2 = c, leaveFromClass: p3 = `${n2}-leave-from`, leaveActiveClass: d2 = `${n2}-leave-active`, leaveToClass: R3 = `${n2}-leave-to` } = t2, P2 = Ge4(o), le2 = P2 && P2[0], ue3 = P2 && P2[1], { onBeforeEnter: at2, onEnter: lt3, onEnterCancelled: ut, onLeave: ft2, onLeaveCancelled: fe3, onBeforeAppear: pe3 = at2, onAppear: de4 = lt3, onAppearCancelled: he3 = ut } = e, q2 = (u, h2, N2, H2) => {
      u._enterCancelled = H2, v3(u, h2 ? f2 : c), v3(u, h2 ? l2 : r2), N2 && N2();
    }, pt3 = (u, h2) => {
      u._isLeaving = false, v3(u, p3), v3(u, R3), v3(u, d2), h2 && h2();
    }, dt3 = (u) => (h2, N2) => {
      let H2 = u ? de4 : lt3, ht2 = () => q2(h2, u, N2);
      T(H2, [h2, ht2]), vt2(() => {
        v3(h2, u ? a : i), g3(h2, u ? f2 : c), St3(H2) || Et3(h2, s, le2, ht2);
      });
    };
    return de(e, { onBeforeEnter(u) {
      T(at2, [u]), g3(u, i), g3(u, r2);
    }, onBeforeAppear(u) {
      T(pe3, [u]), g3(u, a), g3(u, l2);
    }, onEnter: dt3(false), onAppear: dt3(true), onLeave(u, h2) {
      u._isLeaving = true;
      let N2 = () => pt3(u, h2);
      g3(u, p3), u._enterCancelled ? (g3(u, d2), nt2(u)) : (nt2(u), g3(u, d2)), vt2(() => {
        u._isLeaving && (v3(u, p3), g3(u, R3), St3(ft2) || Et3(u, s, ue3, N2));
      }), T(ft2, [u, N2]);
    }, onEnterCancelled(u) {
      q2(u, false, void 0, true), T(ut, [u]);
    }, onAppearCancelled(u) {
      q2(u, true, void 0, true), T(he3, [u]);
    }, onLeaveCancelled(u) {
      pt3(u), T(fe3, [u]);
    } });
  }
  function Ge4(t2) {
    if (t2 == null) return null;
    if (f(t2)) return [Y3(t2.enter), Y3(t2.leave)];
    {
      let e = Y3(t2);
      return [e, e];
    }
  }
  function Y3(t2) {
    return Ce(t2);
  }
  function g3(t2, e) {
    e.split(/\s+/).forEach((n2) => n2 && t2.classList.add(n2)), (t2[M3] || (t2[M3] = /* @__PURE__ */ new Set())).add(e);
  }
  function v3(t2, e) {
    e.split(/\s+/).forEach((s) => s && t2.classList.remove(s));
    let n2 = t2[M3];
    n2 && (n2.delete(e), n2.size || (t2[M3] = void 0));
  }
  function vt2(t2) {
    requestAnimationFrame(() => {
      requestAnimationFrame(t2);
    });
  }
  var qe2 = 0;
  function Et3(t2, e, n2, s) {
    let o = t2._endId = ++qe2, i = () => {
      o === t2._endId && s();
    };
    if (n2 != null) return setTimeout(i, n2);
    let { type: r2, timeout: c, propCount: a } = Xt2(t2, e);
    if (!r2) return s();
    let l2 = r2 + "end", f2 = 0, p3 = () => {
      t2.removeEventListener(l2, d2), i();
    }, d2 = (R3) => {
      R3.target === t2 && ++f2 >= a && p3();
    };
    setTimeout(() => {
      f2 < a && p3();
    }, c + 1), t2.addEventListener(l2, d2);
  }
  function Xt2(t2, e) {
    let n2 = window.getComputedStyle(t2), s = (P2) => (n2[P2] || "").split(", "), o = s(`${S2}Delay`), i = s(`${S2}Duration`), r2 = Ct2(o, i), c = s(`${L2}Delay`), a = s(`${L2}Duration`), l2 = Ct2(c, a), f2 = null, p3 = 0, d2 = 0;
    e === S2 ? r2 > 0 && (f2 = S2, p3 = r2, d2 = i.length) : e === L2 ? l2 > 0 && (f2 = L2, p3 = l2, d2 = a.length) : (p3 = Math.max(r2, l2), f2 = p3 > 0 ? r2 > l2 ? S2 : L2 : null, d2 = f2 ? f2 === S2 ? i.length : a.length : 0);
    let R3 = f2 === S2 && /\b(?:transform|all)(?:,|$)/.test(s(`${S2}Property`).toString());
    return { type: f2, timeout: p3, propCount: d2, hasTransform: R3 };
  }
  function Ct2(t2, e) {
    for (; t2.length < e.length; ) t2 = t2.concat(t2);
    return Math.max(...e.map((n2, s) => wt2(n2) + wt2(t2[s])));
  }
  function wt2(t2) {
    return t2 === "auto" ? 0 : Number(t2.slice(0, -1).replace(",", ".")) * 1e3;
  }
  function nt2(t2) {
    return (t2 ? t2.ownerDocument : document).body.offsetHeight;
  }
  function Xe4(t2, e, n2) {
    let s = t2[M3];
    s && (e = (e ? [e, ...s] : [...s]).join(" ")), e == null ? t2.removeAttribute("class") : n2 ? t2.setAttribute("class", e) : t2.className = e;
  }
  var K3 = /* @__PURE__ */ Symbol("_vod");
  var ct3 = /* @__PURE__ */ Symbol("_vsh");
  var Yt2 = /* @__PURE__ */ Symbol("");
  var Ze2 = /(?:^|;)\s*display\s*:/;
  function Qe3(t2, e, n2) {
    let s = t2.style, o = p(n2), i = false;
    if (n2 && !o) {
      if (e) if (p(e)) for (let r2 of e.split(";")) {
        let c = r2.slice(0, r2.indexOf(":")).trim();
        n2[c] == null && I(s, c, "");
      }
      else for (let r2 in e) n2[r2] == null && I(s, r2, "");
      for (let r2 in n2) {
        r2 === "display" && (i = true);
        let c = n2[r2];
        c != null ? en2(t2, r2, !p(e) && e ? e[r2] : void 0, c) || I(s, r2, c) : I(s, r2, "");
      }
    } else if (o) {
      if (e !== n2) {
        let r2 = s[Yt2];
        r2 && (n2 += ";" + r2), s.cssText = n2, i = Ze2.test(n2);
      }
    } else e && t2.removeAttribute("style");
    K3 in t2 && (t2[K3] = i ? s.display : "", t2[ct3] && (s.display = "none"));
  }
  var F3 = /\s*!important$/;
  function I(t2, e, n2) {
    if (d(n2)) n2.forEach((s) => I(t2, e, s));
    else if (n2 == null && (n2 = ""), e.startsWith("--")) F3.test(n2) ? t2.setProperty(e, n2.replace(F3, ""), "important") : t2.setProperty(e, n2);
    else {
      let s = tn(t2, e);
      F3.test(n2) ? t2.setProperty(H(s), n2.replace(F3, ""), "important") : t2[s] = n2;
    }
  }
  var Nt3 = ["Webkit", "Moz", "ms"];
  var J3 = {};
  function tn(t2, e) {
    let n2 = J3[e];
    if (n2) return n2;
    let s = Ae(e);
    if (s !== "filter" && s in t2) return J3[e] = s;
    s = B(s);
    for (let o = 0; o < Nt3.length; o++) {
      let i = Nt3[o] + s;
      if (i in t2) return J3[e] = i;
    }
    return e;
  }
  function en2(t2, e, n2, s) {
    return t2.tagName === "TEXTAREA" && (e === "width" || e === "height") && p(s) && n2 === s;
  }
  var Tt2 = "http://www.w3.org/1999/xlink";
  function At3(t2, e, n2, s, o, i = Ye(e)) {
    s && e.startsWith("xlink:") ? n2 == null ? t2.removeAttributeNS(Tt2, e.slice(6, e.length)) : t2.setAttributeNS(Tt2, e, n2) : n2 == null || i && !Ke(n2) ? t2.removeAttribute(e) : t2.setAttribute(e, i ? "" : y(n2) ? String(n2) : n2);
  }
  function Vt(t2, e, n2, s, o) {
    if (e === "innerHTML" || e === "textContent") {
      n2 != null && (t2[e] = e === "innerHTML" ? Wt(n2) : n2);
      return;
    }
    let i = t2.tagName;
    if (e === "value" && i !== "PROGRESS" && !i.includes("-")) {
      let c = i === "OPTION" ? t2.getAttribute("value") || "" : t2.value, a = n2 == null ? t2.type === "checkbox" ? "on" : "" : String(n2);
      (c !== a || !("_value" in t2)) && (t2.value = a), n2 == null && t2.removeAttribute(e), t2._value = n2;
      return;
    }
    let r2 = false;
    if (n2 === "" || n2 == null) {
      let c = typeof t2[e];
      c === "boolean" ? n2 = Ke(n2) : n2 == null && c === "string" ? (n2 = "", r2 = true) : c === "number" && (n2 = 0, r2 = true);
    }
    try {
      t2[e] = n2;
    } catch {
    }
    r2 && t2.removeAttribute(o || e);
  }
  function b3(t2, e, n2, s) {
    t2.addEventListener(e, n2, s);
  }
  function nn2(t2, e, n2, s) {
    t2.removeEventListener(e, n2, s);
  }
  var Ot2 = /* @__PURE__ */ Symbol("_vei");
  function sn2(t2, e, n2, s, o = null) {
    let i = t2[Ot2] || (t2[Ot2] = {}), r2 = i[e];
    if (s && r2) r2.value = s;
    else {
      let [c, a] = cn2(e);
      if (s) {
        let l2 = i[e] = un2(s, o);
        b3(t2, c, l2, a);
      } else r2 && (nn2(t2, c, r2, a), i[e] = void 0);
    }
  }
  var on2 = /(Once|Passive|Capture)$/;
  var rn2 = /^on:?(?:Once|Passive|Capture)$/;
  function cn2(t2) {
    let e, n2;
    for (; (n2 = t2.match(on2)) && !rn2.test(t2); ) e || (e = {}), t2 = t2.slice(0, t2.length - n2[1].length), e[n2[1].toLowerCase()] = true;
    return [t2[2] === ":" ? t2.slice(3) : H(t2.slice(2)), e];
  }
  var Z2 = 0;
  var an2 = Promise.resolve();
  var ln2 = () => Z2 || (an2.then(() => Z2 = 0), Z2 = Date.now());
  function un2(t2, e) {
    let n2 = (s) => {
      if (!s._vts) s._vts = Date.now();
      else if (s._vts <= n2.attached) return;
      let o = n2.value;
      if (d(o)) {
        let i = s.stopImmediatePropagation;
        s.stopImmediatePropagation = () => {
          i.call(s), s._stopped = true;
        };
        let r2 = o.slice(), c = [s];
        for (let a = 0; a < r2.length && !s._stopped; a++) {
          let l2 = r2[a];
          l2 && Ie2(l2, e, 5, c);
        }
      } else Ie2(o, e, 5, [s]);
    };
    return n2.value = t2, n2.attached = ln2(), n2;
  }
  var Rt2 = (t2) => t2.charCodeAt(0) === 111 && t2.charCodeAt(1) === 110 && t2.charCodeAt(2) > 96 && t2.charCodeAt(2) < 123;
  var fn = (t2, e, n2, s, o, i) => {
    let r2 = o === "svg";
    e === "class" ? Xe4(t2, s, r2) : e === "style" ? Qe3(t2, n2, s) : pe(e) ? fe(e) || sn2(t2, e, n2, s, i) : (e[0] === "." ? (e = e.slice(1), true) : e[0] === "^" ? (e = e.slice(1), false) : pn(t2, e, s, r2)) ? (Vt(t2, e, s), !t2.tagName.includes("-") && (e === "value" || e === "checked" || e === "selected") && At3(t2, e, s, r2, i, e !== "value")) : t2._isVueCE && (dn2(t2, e) || t2._def.__asyncLoader && (/[A-Z]/.test(e) || !p(s))) ? Vt(t2, Ae(e), s, i, e) : (e === "true-value" ? t2._trueValue = s : e === "false-value" && (t2._falseValue = s), At3(t2, e, s, r2));
  };
  function pn(t2, e, n2, s) {
    if (s) return !!(e === "innerHTML" || e === "textContent" || e in t2 && Rt2(e) && g(n2));
    if (e === "spellcheck" || e === "draggable" || e === "translate" || e === "autocorrect" || e === "sandbox" && t2.tagName === "IFRAME" || e === "form" || e === "list" && t2.tagName === "INPUT" || e === "type" && t2.tagName === "TEXTAREA") return false;
    if (e === "width" || e === "height") {
      let o = t2.tagName;
      if (o === "IMG" || o === "VIDEO" || o === "CANVAS" || o === "SOURCE") return false;
    }
    return Rt2(e) && p(n2) ? false : e in t2;
  }
  function dn2(t2, e) {
    let n2 = t2._def.props;
    if (!n2) return false;
    let s = Ae(e);
    return Array.isArray(n2) ? n2.some((o) => Ae(o) === s) : Object.keys(n2).some((o) => Ae(o) === s);
  }
  var Jt3 = /* @__PURE__ */ new WeakMap();
  var Zt2 = /* @__PURE__ */ new WeakMap();
  var W3 = /* @__PURE__ */ Symbol("_moveCb");
  var Mt3 = /* @__PURE__ */ Symbol("_enterCb");
  var gn2 = (t2) => (delete t2.props.mode, t2);
  var yn2 = gn2({ name: "TransitionGroup", props: de({}, Gt3, { tag: String, moveClass: String }), setup(t2, { slots: e }) {
    let n2 = Me2(), s = Ps(), o, i;
    return oo(() => {
      if (!o.length) return;
      let r2 = t2.moveClass || `${t2.name || "v"}-move`;
      if (!En2(o[0].el, n2.vnode.el, r2)) {
        o = [];
        return;
      }
      o.forEach(bn), o.forEach(Sn);
      let c = o.filter(vn);
      nt2(n2.vnode.el), c.forEach((a) => {
        let l2 = a.el, f2 = l2.style;
        g3(l2, r2), f2.transform = f2.webkitTransform = f2.transitionDuration = "";
        let p3 = l2[W3] = (d2) => {
          d2 && d2.target !== l2 || (!d2 || d2.propertyName.endsWith("transform")) && (l2.removeEventListener("transitionend", p3), l2[W3] = null, v3(l2, r2));
        };
        l2.addEventListener("transitionend", p3);
      }), o = [];
    }), () => {
      let r2 = p2(t2), c = qt3(r2), a = r2.tag || ge3;
      if (o = [], i) for (let l2 = 0; l2 < i.length; l2++) {
        let f2 = i[l2];
        f2.el && f2.el instanceof Element && !f2.el[ct3] && (o.push(f2), gt2(f2, Rn(f2, c, s, n2)), Jt3.set(f2, Qt3(f2.el)));
      }
      i = e.default ? eo(e.default()) : [];
      for (let l2 = 0; l2 < i.length; l2++) {
        let f2 = i[l2];
        f2.key != null && gt2(f2, Rn(f2, c, s, n2));
      }
      return se3(a, null, i);
    };
  } });
  function bn(t2) {
    let e = t2.el;
    e[W3] && e[W3](), e[Mt3] && e[Mt3]();
  }
  function Sn(t2) {
    Zt2.set(t2, Qt3(t2.el));
  }
  function vn(t2) {
    let e = Jt3.get(t2), n2 = Zt2.get(t2), s = e.left - n2.left, o = e.top - n2.top;
    if (s || o) {
      let i = t2.el, r2 = i.style, c = i.getBoundingClientRect(), a = 1, l2 = 1;
      return i.offsetWidth && (a = c.width / i.offsetWidth), i.offsetHeight && (l2 = c.height / i.offsetHeight), (!Number.isFinite(a) || a === 0) && (a = 1), (!Number.isFinite(l2) || l2 === 0) && (l2 = 1), Math.abs(a - 1) < 0.01 && (a = 1), Math.abs(l2 - 1) < 0.01 && (l2 = 1), r2.transform = r2.webkitTransform = `translate(${s / a}px,${o / l2}px)`, r2.transitionDuration = "0s", t2;
    }
  }
  function Qt3(t2) {
    let e = t2.getBoundingClientRect();
    return { left: e.left, top: e.top };
  }
  function En2(t2, e, n2) {
    let s = t2.cloneNode(), o = t2[M3];
    o && o.forEach((c) => {
      c.split(/\s+/).forEach((a) => a && s.classList.remove(a));
    }), n2.split(/\s+/).forEach((c) => c && s.classList.add(c)), s.style.display = "none";
    let i = e.nodeType === 1 ? e : e.parentNode;
    i.appendChild(s);
    let { hasTransform: r2 } = Xt2(s);
    return i.removeChild(s), r2;
  }
  var oe4 = de({ patchProp: fn }, We4);
  var j2;
  function ie3() {
    return j2 || (j2 = Kl(oe4));
  }
  var It2 = ((...t2) => {
    let e = ie3().createApp(...t2), { mount: n2 } = e;
    return e.mount = (s) => {
      let o = ae3(s);
      if (!o) return;
      let i = e._component;
      !g(i) && !i.render && !i.template && (i.template = o.innerHTML), o.nodeType === 1 && (o.textContent = "");
      let r2 = n2(o, false, ce4(o));
      return o instanceof Element && (o.removeAttribute("v-cloak"), o.setAttribute("data-v-app", "")), r2;
    }, e;
  });
  function ce4(t2) {
    if (t2 instanceof SVGElement) return "svg";
    if (typeof MathMLElement == "function" && t2 instanceof MathMLElement) return "mathml";
  }
  function ae3(t2) {
    return p(t2) ? document.querySelector(t2) : t2;
  }

  // pkg/jsengine/scripts/choysummount/choysummount.js
  function normalizeStubs(stubs) {
    if (!stubs) return {};
    if (stubs === true) return { __all: true };
    var out = /* @__PURE__ */ Object.create(null);
    Object.keys(stubs).forEach(function(name) {
      var val = stubs[name];
      if (val === true) {
        out[name] = {
          name,
          render: function() {
            return zl(name);
          }
        };
      } else if (val && typeof val === "object") {
        out[name] = val;
      }
    });
    return out;
  }
  function installStubs(app, stubs) {
    var map = normalizeStubs(stubs);
    Object.keys(map).forEach(function(name) {
      if (name === "__all") return;
      app.component(name, map[name]);
    });
  }
  function makeWrapper(app, el, vm) {
    return {
      element: el,
      vm,
      find: function(sel) {
        var node = el.querySelector(sel);
        return {
          exists: function() {
            return !!node;
          },
          element: node,
          text: function() {
            return node ? String(node.textContent || "") : "";
          },
          trigger: function(eventName) {
            if (!node) return Promise.resolve();
            var Evt = typeof Event === "function" ? Event : null;
            var evt = Evt ? new Evt(String(eventName), { bubbles: true }) : { type: String(eventName) };
            if (typeof node.dispatchEvent === "function") {
              node.dispatchEvent(evt);
            }
            return flushPromises();
          }
        };
      },
      trigger: function(eventName) {
        var Evt = typeof Event === "function" ? Event : null;
        var evt = Evt ? new Evt(String(eventName), { bubbles: true }) : { type: String(eventName) };
        if (el && typeof el.dispatchEvent === "function") {
          el.dispatchEvent(evt);
        }
        return flushPromises();
      },
      text: function() {
        return el ? String(el.textContent || "") : "";
      },
      html: function() {
        return el ? String(el.innerHTML || el.textContent || "") : "";
      },
      unmount: function() {
        try {
          app.unmount();
        } catch (_) {
        }
        if (el && el.parentNode) {
          try {
            el.parentNode.removeChild(el);
          } catch (_) {
          }
        }
      }
    };
  }
  function autoStubComponents(components) {
    var auto = /* @__PURE__ */ Object.create(null);
    Object.keys(components || {}).forEach(function(name) {
      auto[name] = {
        name,
        render: function() {
          return zl("div", { class: "stub-" + name }, name);
        }
      };
    });
    return auto;
  }
  function withComponentStubs(component, stubs) {
    var map = normalizeStubs(stubs);
    var base = component && typeof component === "object" ? component : {};
    if (map.__all) {
      map = Object.assign(autoStubComponents(base.components), map);
      delete map.__all;
    }
    var keys = Object.keys(map);
    if (!keys.length) {
      return component;
    }
    var merged = Object.assign({}, base);
    merged.components = Object.assign({}, base.components || {}, map);
    return merged;
  }
  function mount(component, options) {
    options = options || {};
    var doc = globalThis.document;
    if (!doc || typeof doc.createElement !== "function") {
      throw new Error("choysumMount: minimal DOM not installed (call InstallMinimalDOM first)");
    }
    var el = doc.createElement("div");
    if (doc.body && typeof doc.body.appendChild === "function") {
      doc.body.appendChild(el);
    }
    var root = withComponentStubs(component, options.stubs);
    var app = It2(root, options.props || {});
    installStubs(app, options.stubs);
    if (typeof options.shallow === "boolean" && options.shallow) {
      if (options.stubs === true) {
        app.config.warnHandler = function() {
        };
      }
    }
    var vm = app.mount(el);
    return makeWrapper(app, el, vm);
  }
  function shallowMount(component, options) {
    options = options || {};
    if (options.stubs == null) {
      options = Object.assign({}, options, { stubs: true, shallow: true });
    } else {
      options = Object.assign({}, options, { shallow: true });
    }
    return mount(component, options);
  }
  function flushPromises() {
    var chain = Promise.resolve();
    if (typeof ms === "function") {
      chain = chain.then(function() {
        return ms();
      });
    }
    return chain.then(function() {
      return new Promise(function(resolve) {
        if (typeof queueMicrotask === "function") {
          queueMicrotask(resolve);
        } else {
          Promise.resolve().then(resolve);
        }
      });
    }).then(function() {
      return new Promise(function(resolve) {
        if (typeof queueMicrotask === "function") {
          queueMicrotask(resolve);
        } else {
          Promise.resolve().then(resolve);
        }
      });
    });
  }
  var api = { mount, shallowMount, flushPromises };
  if (typeof globalThis !== "undefined") {
    globalThis.choysumMount = api;
  }

  // sfc-script:/Users/wangbuke/choysum/internal/testing/frontend/testdata/vue_host/HostCounter.vue?type=script
  var marker = 42;
  var HostCounter_default = /* @__PURE__ */ Is({
    __name: "HostCounter",
    setup(__props, { expose: __expose }) {
      __expose();
      const count = Ot(0);
      function bump() {
        count.value += 1;
      }
      const __returned__ = { count, marker, bump };
      Object.defineProperty(__returned__, "__isScriptSetup", { enumerable: false, value: true });
      return __returned__;
    }
  });

  // sfc-template:/Users/wangbuke/choysum/internal/testing/frontend/testdata/vue_host/HostCounter.vue?type=template
  function render(_ctx, _cache, $props, $setup, $data, $options) {
    return sn(), Jl(
      ge3,
      null,
      [
        Co(
          "button",
          {
            class: "bump",
            type: "button",
            onClick: $setup.bump
          },
          ie($setup.count),
          1
          /* TEXT */
        ),
        Co("span", { class: "marker" }, ie($setup.marker))
      ],
      64
      /* STABLE_FRAGMENT */
    );
  }

  // internal/testing/frontend/testdata/vue_host/HostCounter.vue
  HostCounter_default.render = render;
  HostCounter_default.__file = "/Users/wangbuke/choysum/internal/testing/frontend/testdata/vue_host/HostCounter.vue";
  var HostCounter_default2 = HostCounter_default;

  // internal/testing/frontend/testdata/vue_host/entry_mount.ts
  var w3 = mount(HostCounter_default2);
  var btn = w3.find(".bump");
  var marker2 = w3.find(".marker");
  globalThis.__hostResult = {
    hasBtn: btn.exists(),
    markerText: marker2.text(),
    btnTextBefore: btn.text()
  };
  btn.trigger("click").then(() => {
    globalThis.__hostResult.btnTextAfter = w3.find(".bump").text();
    globalThis.__hostResult.ready = true;
  });
})();
/*! Bundled license information:

@vue/shared/dist/shared.esm-bundler.js:
  (**
  * @vue/shared v3.5.42
  * (c) 2018-present Yuxi (Evan) You and Vue contributors
  * @license MIT
  **)
*/
/*! Bundled license information:

@vue/reactivity/dist/reactivity.esm-bundler.js:
  (**
  * @vue/reactivity v3.5.42
  * (c) 2018-present Yuxi (Evan) You and Vue contributors
  * @license MIT
  **)
*/
/*! Bundled license information:

@vue/runtime-core/dist/runtime-core.esm-bundler.js:
  (**
  * @vue/runtime-core v3.5.42
  * (c) 2018-present Yuxi (Evan) You and Vue contributors
  * @license MIT
  **)
*/
/*! Bundled license information:

@vue/runtime-dom/dist/runtime-dom.esm-bundler.js:
  (**
  * @vue/runtime-dom v3.5.42
  * (c) 2018-present Yuxi (Evan) You and Vue contributors
  * @license MIT
  **)
*/
/*! Bundled license information:

vue/dist/vue.runtime.esm-bundler.js:
  (**
  * vue v3.5.42
  * (c) 2018-present Yuxi (Evan) You and Vue contributors
  * @license MIT
  **)
*/
//# sourceMappingURL=vue-host.bundle.js.map

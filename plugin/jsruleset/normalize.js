(function normalize(root) {
  const active = new Set();
  const enabledSeverities = new WeakMap();
  const ruleKeys = new Set(['description','documentationUrl','message','given','formats','resolved','recommended','type','severity','then','howToFix','autoFixFunction']);
  const setKeys = new Set(['rules','extends','formats','aliases','description','documentationUrl']);
  function checkKeys(value, allowed, where) {
    for (const key of Object.keys(value)) if (!allowed.has(key)) throw Error('Unsupported '+where+' property: '+key);
  }
  function severity(value) {
    if (typeof value === 'number') {
      if (!Number.isInteger(value) || value < -1 || value > 3) throw Error('Invalid severity: '+value);
      return ['off','error','warn','info','hint'][value+1];
    }
    return value;
  }
  function formats(value) {
    if (!Array.isArray(value)) throw Error('formats must be an array');
    return value.map(f => {
      if (typeof f === 'function') return __register(f, 'format');
      if (typeof f !== 'string') throw Error('format must be a predicate or string');
      return f;
    });
  }
  function action(value) {
    if (!value || typeof value !== 'object' || Array.isArray(value)) throw Error('then must contain actions');
    checkKeys(value, new Set(['function','field','functionOptions']), 'action');
    const out = {...value};
    if (typeof value.function === 'function') {
      if (value.function.__validateOptions) value.function.__validateOptions(value.functionOptions);
      out.function = value.function.__nativeName || __register(value.function,'function');
    } else if (typeof value.function !== 'string') throw Error('then.function must be callable or a name');
    return out;
  }
  function rule(value, inherited) {
    if (typeof value === 'boolean' || typeof value === 'string' || typeof value === 'number') {
      if (!inherited || typeof inherited!=='object') return value===true && typeof inherited==='string' && inherited!=='off' ? inherited : typeof value==='number'?severity(value):value;
      const enabled=enabledSeverities.get(inherited) || (inherited.severity==='off'?'warn':inherited.severity);
      const out={...inherited,severity:value===true?enabled:value===false?'off':severity(value)};
      enabledSeverities.set(out,out.severity==='off'?enabled:out.severity);return out;
    }
    if (!value || typeof value !== 'object' || Array.isArray(value)) throw Error('Invalid rule definition');
    checkKeys(value,ruleKeys,'rule');
    const out = {resolved:true, recommended:true, severity:'warn', type:'validation', ...inherited, ...value};
    out.severity=severity(out.severity);
    enabledSeverities.set(out,out.severity==='off'?(inherited && enabledSeverities.get(inherited)) || 'warn':out.severity);
    if (out.formats) out.formats=formats(out.formats);
    if (out.then) out.then=Array.isArray(out.then)?out.then.map(action):action(out.then);
    return out;
  }
  function visit(value) {
    if (!value || typeof value !== 'object' || Array.isArray(value) || typeof value.then === 'function') throw Error('Ruleset export must be an object, not a factory or Promise');
    if (active.has(value)) throw Error('Circular ruleset inheritance');
    active.add(value);checkKeys(value,setKeys,'ruleset');
    const out={rules:{},aliases:{},extends:[]};
    let parents=value.extends===undefined?[]:value.extends;
    if (!Array.isArray(parents)) parents=[parents];
    for (let parent of parents) {
      let mode='recommended';
      if (Array.isArray(parent)) { if(parent.length!==2) throw Error('extends tuple requires two entries'); [parent,mode]=parent; }
      if (!['all','recommended','off'].includes(mode)) throw Error('Unsupported extends mode: '+mode);
      if (typeof parent==='string') { out.extends.push([parent,mode]); continue; }
      const base=visit(parent);
      Object.assign(out.aliases,base.aliases);out.extends.push(...base.extends);
      for(const id of Object.keys(base.rules)) {
        const r=base.rules[id];if(typeof r!=='object'){out.rules[id]=mode==='off'?false:r;continue;}out.rules[id]={...r};enabledSeverities.set(out.rules[id],enabledSeverities.get(r) || r.severity);
        if(mode==='off'||(mode==='recommended'&&r.recommended===false)) out.rules[id].severity='off';
      }
    }
    if(value.aliases) {
      for(const name of Object.keys(value.aliases).sort()) {
        const alias=value.aliases[name];
        if(alias && !Array.isArray(alias) && typeof alias==='object') {
          checkKeys(alias,new Set(['description','targets']),'alias');
          out.aliases[name]={...alias,targets:alias.targets.map(t=>({...t,formats:formats(t.formats)}))};
        } else out.aliases[name]=alias;
      }
    }
    const defaults=value.formats ? formats(value.formats) : undefined;
    for(const id of Object.keys(value.rules||{}).sort()) {
      const inherited=out.rules[id];
      const next=rule(value.rules[id],inherited);
      if(next && typeof next==='object' && !next.formats && defaults) next.formats=defaults;
      out.rules[id]=next;
    }
    for(const key of ['description','documentationUrl']) if(value[key]!==undefined) out[key]=value[key];
    active.delete(value);return out;
  }
  return JSON.stringify(visit(root));
})

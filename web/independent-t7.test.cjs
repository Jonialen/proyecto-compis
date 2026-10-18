const {test}=require('node:test');
const a=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
const code=fs.readFileSync(require('node:path').join(__dirname,'app.js'),'utf8');
function page(){
 const nodes=new Map(),sinks=[],timers=[],readers=[];
 function el(){let html='';return {textContent:'',children:[],listeners:{},style:{},disabled:false,classList:{add(){},remove(){},toggle(){}},setAttribute(){},removeAttribute(){},focus(){},refresh(){},reset(){},remove(){},
 get innerHTML(){return html},set innerHTML(v){html=v;sinks.push(v)},appendChild(n){this.children.push(n)},replaceChildren(...ns){this.children=ns;html='';this.textContent=''},addEventListener(k,f){this.listeners[k]=f},click(){return this.listeners.click?.()}}}
 const get=id=>{if(!nodes.has(id))nodes.set(id,el());return nodes.get(id)};
 const c=vm.createContext({console,document:{getElementById:get,createElement:el,addEventListener(){},querySelectorAll:s=>s.startsWith('#sidebar')?[{value:'ll1'}]:[]},setTimeout:(f,d)=>timers.push([d,f]),FileReader:class{constructor(){readers.push(this)}readAsText(){}},fetch:async()=>({ok:true,json:async()=>report('ok')})});
 vm.runInContext(code,c);
 vm.runInContext(`for(const k of ['cps','yal','yalp','input']){let v=k==='yalp'?'%token INT\\n%%\\ns : INT ;':'let n = 1;',on=[];state.editors[k]={getValue:()=>v,setValue:x=>{v=x;on.forEach(f=>f())},on:(e,f)=>on.push(f),refresh(){},focus(){}}}`,c);
 const run=s=>vm.runInContext(s,c); const text=n=>[n.textContent,...n.children.map(text)].join('\n');
 return {c,get,run,text,sinks,timers,readers,upload(target,name,type='text/plain'){get('load-'+target).click();get('file-input').files=[{name,type}];get('file-input').listeners.change();return readers.at(-1)}};
}
function report(t){return {tac:{instructions:[]},tacText:t,diagnostics:[],scopes:[]}}
function delay(){let resolve,reject;const promise=new Promise((r,j)=>{resolve=r;reject=j});return {promise,resolve,reject}}
const attack='<svg/onload=globalThis.pwned=1>&"\'🧪';
test('S9 text boundary: every compiler field and filename/error is inert',async()=>{
 const p=page();p.run('bindSourceChanges();bindSidebar();bindRunButton()');
 p.run('state.editors.cps.setValue('+JSON.stringify(attack)+')');
 const r={...report(attack),ast:{kind:attack,label:attack},diagnostics:[{phase:attack,code:attack,message:attack,span:{start:{line:1,column:2}}}],scopes:[{id:1,kind:attack,symbols:[{kind:attack,name:attack,type:{name:attack}}]}],codegen:{unit:attack,symbols:[{id:attack,scopeId:1,symbol:{kind:attack,name:attack,type:{name:attack},span:attack},locations:[{space:attack,offset:0,size:1,role:attack}],labels:[attack]}],spaces:[{id:attack,slots:[{type:attack}]}],labels:[{name:attack,kind:attack,owner:attack}]}};
 p.c.fetch=async()=>({ok:true,json:async()=>r});await p.run('runCompiscript("compile")');
 for(const id of ['source','ast','diagnostics','environments','tac','symbols'])a.ok(p.text(p.get('rp-'+id)).includes(attack),id);
 a.equal(p.sinks.length,0);a.equal(p.run('globalThis.pwned'),undefined);
 const reader=p.upload('cps',attack+'.cps');reader.onload({target:{result:attack}});
 a.equal(p.get('status-text').textContent,'Loaded: '+attack+'.cps');
 p.c.fetch=async()=>{throw Error(attack)};await p.run('runCompiscript("compile")');
 a.equal(p.get('status-text').textContent,'Error: '+attack);a.equal(p.sinks.length,0);
});
for(const invalidate of ['edit','clear','sample','feature']) for(const outcome of ['success','failure','reject'])test('S9 stale JSON '+invalidate+'/'+outcome,async()=>{
 const p=page();p.run('bindSourceChanges();bindRunButton();switchEditorTab("cps")');
 const body=delay();p.c.fetch=async()=>({ok:outcome!=='failure',json:()=>body.promise});const old=p.run('runCompiscript("compile")');await Promise.resolve();
 if(invalidate==='edit')p.run('state.editors.cps.setValue("let x=2;")');
 if(invalidate==='clear')p.get('btn-clear-cps').click();if(invalidate==='sample')p.get('btn-sample-cps').click();if(invalidate==='feature')p.run('switchEditorTab("yalp")');
 const fresh=delay();p.c.fetch=()=>fresh.promise;
 const next=p.run(invalidate==='feature'?'runAnalysis()':'runCompiscript("compile")');
 if(outcome==='reject')body.reject(Error(attack));else body.resolve(outcome==='failure'?{error:attack}:report('OLD'));
 await old;a.equal(p.get('btn-run').disabled,true);a.equal(p.get('status-text').className,'loading');
 fresh.resolve({ok:true,json:async()=>invalidate==='feature'?{success:true,methods:{ll1:{accepted:true}}}:report('NEW')});await next;
 a.equal(p.get('btn-run').disabled,false);a.equal(p.get('status-text').className,'ok');a.ok(!p.text(p.get('rp-tac')).includes('OLD'));
});
test('S9 upload generations, failures, empty source and sample',async()=>{
 const p=page();p.run('bindSourceChanges();bindSidebar();bindRunButton()');
 const old=p.upload('cps','old.cps'),fresh=p.upload('cps','new.CPS','application/octet-stream');
 fresh.onload({target:{result:''}});old.onload({target:{result:'OLD'}});old.onerror();
 a.equal(p.run('state.editors.cps.getValue()'),'');a.equal(p.get('status-text').textContent,'Loaded: new.CPS');
 for(const action of ['btn-clear-cps','btn-sample-cps']){const late=p.upload('cps','late.cps');p.get(action).click();const v=p.run('state.editors.cps.getValue()');late.onload({target:{result:'OLD'}});a.equal(p.run('state.editors.cps.getValue()'),v)}
 let sent;p.c.fetch=async(u,o)=>{sent=JSON.parse(o.body);return {ok:true,json:async()=>report('SAMPLE')}};await p.run('runCompiscript("compile")');a.match(sent.source,/foreach/);
 const bad=p.readers.length;p.upload('cps','wrong.txt');a.equal(p.readers.length,bad);a.equal(p.get('status-text').textContent,'Please select a plain-text .cps file.');
});
test('S9 modal generation cannot start after newer failed compile or clear',async()=>{
 const p=page();p.run('bindSourceChanges();bindRunButton();processPayment()');
 let calls=0;p.c.fetch=async()=>{calls++;throw Error('offline')};p.run('switchEditorTab("cps")');await p.run('runCompiscript("compile")');p.get('btn-clear-cps').click();p.run('switchEditorTab("yalp")');
 for(const [d,f] of p.timers)if(d===3200)f();await new Promise(setImmediate);a.equal(calls,1);a.equal(p.get('btn-run').disabled,false);
});

'use strict';
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const html = fs.readFileSync(path.join(__dirname, '..', 'static', 'settings.html'), 'utf8');
const match = html.match(/<script>([\s\S]*?)<\/script>/);
if (!match) throw new Error('Missing embedded Settings script');
const code = match[1];

test('Settings JavaScript parses in supported browsers', () => {
  assert.doesNotThrow(() => new vm.Script(code, {filename: 'settings.html'}));
});

test('Settings initialisation renders fields and save applies CSRF-protected changes', async () => {
  const elems = new Map();
  const $ = id => {
    if (!elems.has(id)) elems.set(id, {id, value: '', checked:false, textContent:'', hidden:false, disabled:false, onclick:null, addEventListener(type,fn){this['on'+type] = fn}});
    return elems.get(id);
  };
  const initial = {
    enabled:false,lan:false,discovery:false,streamMode:'redirect',baseUrl:'http://127.0.0.1:45782',
    region:'Melbourne',defaultLanUrl:'http://192.168.1.20:45782',
    m3u:'',xmltv:'',status:'Disabled',managed:false,csrf:'test-csrf'
  };
  const posts=[];
  const ctx={ document:{getElementById:$},confirm:()=>true,console,
    fetch: async (url,opts) => {
      if (opts?.method !== 'POST') return {ok:true,json:async()=>initial};
      posts.push({url,opts});
      return {ok:true,json:async()=>({...initial,enabled:true,lan:true,baseUrl:'http://192.168.1.20:45782',status:'Listening on 0.0.0.0:45782',firewallMessage:'Private LAN firewall rule configured.'})};
    }
  };
  vm.runInNewContext(code,ctx,{filename:'settings.html'});
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal($('region').value,'Melbourne');
  assert.equal($('status').textContent,'Current listener: Disabled');
  assert.equal($('feedback').textContent,'Settings loaded.');
  $('lan').checked=true;
  $('lan').onchange();
  assert.equal($('baseUrl').value,'http://192.168.1.20:45782');
  $('enabled').checked=true;
  $('streamMode').value='relay';
  await $('save').onclick();
  assert.equal(posts.length,1);
  assert.equal(posts[0].opts.headers['X-AussieTV-CSRF'],'test-csrf');
  assert.deepEqual(JSON.parse(posts[0].opts.body),{enabled:true,lan:true,discovery:false,streamMode:'relay',baseUrl:'http://192.168.1.20:45782',region:'Melbourne',rotate:false,firewallRepair:false});
  assert.equal($('feedback').textContent,'Private LAN firewall rule configured.');
  await $('repairFirewall').onclick();
  assert.equal(JSON.parse(posts.at(-1).opts.body).firewallRepair,true);
});

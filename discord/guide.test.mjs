import {test} from 'node:test';
import assert from 'node:assert/strict';
import {nowNext, officialLink} from './guide.mjs';
test('programme boundaries and unsorted upcoming listings',()=>{
 const p=[{start:30,end:40,title:'Later'},{start:10,end:20,title:'Now'},{start:20,end:30,title:'Next'}];
 assert.deepEqual(nowNext(p,10),{current:'Now',next:'Next'});
 assert.deepEqual(nowNext(p,20),{current:'Next',next:'Later'});
 assert.equal(nowNext([],10).current,'No current listing');
});
test('only exact official broadcaster links are accepted',()=>{
 assert.equal(officialLink({network:'ABC',official:'https://iview.abc.net.au/collection/959'}),'https://iview.abc.net.au/collection/959');
 assert.equal(officialLink({network:'ABC',official:'https://iview.abc.net.au.evil.invalid/'}),null);
 assert.equal(officialLink({network:'constructor',official:'javascript:alert(1)'}),null);
});

import json,time,threading,http.server,urllib.parse
from pathlib import Path
from playwright.sync_api import sync_playwright
ROOT=Path(__file__).resolve().parents[1]
NOW=int(time.time());
channels=[
 {'id':'abc-tv','name':'ABC TV','number':'21','network':'ABC','logo':'','stream':'','official':'https://iview.abc.net.au', 'programs':[{'title':'Breakfast News','start':NOW-900,'end':NOW+1800},{'title':'Nature','start':NOW+1800,'end':NOW+5400}]},
 {'id':'abc-news','name':'ABC NEWS','number':'24','network':'ABC','logo':'','stream':'','official':'https://iview.abc.net.au', 'programs':[{'title':'ABC News','start':NOW-600,'end':NOW+1200}]},
 {'id':'sbs','name':'SBS','number':'30','network':'SBS','logo':'','stream':'','official':'https://www.sbs.com.au', 'programs':[{'title':'SBS News','start':NOW-1200,'end':NOW+5000}]},
 {'id':'sbs2','name':'SBS2','number':'31','network':'SBS','logo':'','stream':'','official':'https://www.sbs.com.au', 'programs':[{'title':'SBS Two','start':NOW-1200,'end':NOW+5000}]},
 {'id':'seven','name':'Seven','number':'7','network':'Seven','logo':'','stream':'','official':'https://7plus.com.au', 'programs':[{'title':'Seven News','start':NOW-1200,'end':NOW+5000}]},
 {'id':'nine','name':'Channel 9','number':'9','network':'Nine','logo':'','stream':'','official':'https://9now.com.au', 'programs':[{'title':'Nine News','start':NOW-1200,'end':NOW+5000}]},
 {'id':'ten','name':'10','number':'10','network':'Ten','logo':'','stream':'','official':'https://10play.com.au', 'programs':[{'title':'Ten News','start':NOW-1200,'end':NOW+5000}]},
]
class Handler(http.server.BaseHTTPRequestHandler):
 def do_GET(self):
  p=urllib.parse.urlsplit(self.path).path
  if p=='/api/channels':
   body=json.dumps({'channels':channels}).encode();typ='application/json'
  elif p=='/api/artwork':
   body=b'{}';typ='application/json'
  elif p in ('/','/remote.js','/favicon.ico','/app-icon.svg','/aussietv-logo.svg'):
   path='static/index.html' if p=='/' else 'static/'+p[1:]
   body=(ROOT/path).read_bytes();typ='text/html' if p=='/' else 'application/javascript' if p.endswith('js') else 'image/svg+xml'
  else: self.send_error(404);return
  self.send_response(200);self.send_header('Content-Type',typ);self.send_header('Content-Length',str(len(body)));self.end_headers();self.wfile.write(body)
 def log_message(self,*args):pass
server=http.server.ThreadingHTTPServer(('127.0.0.1',0),Handler)
thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
with sync_playwright() as pw:
 import os
 browser=pw.chromium.launch(executable_path=os.environ.get('AUSSIETV_CHROMIUM') or '/usr/bin/chromium',headless=True,args=['--no-sandbox','--disable-gpu'])
 context=browser.new_context(viewport={'width':1280,'height':800})
 page=context.new_page();errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
 import re
 html=(ROOT/'static/index.html').read_text()
 mock="""<script>window.fetch=async function(url){if(String(url).startsWith('/api/channels'))return new Response(JSON.stringify({channels:CHANNELS}),{status:200,headers:{'Content-Type':'application/json'}});if(String(url).startsWith('/api/artwork'))return new Response('{}',{status:200,headers:{'Content-Type':'application/json'}});return new Response('',{status:404})};</script>"""
 mock=mock.replace('CHANNELS',json.dumps(channels))
 html=re.sub(r'<script src="https://cdn\.jsdelivr\.net/[^>]*></script>',mock,html)
 html=html.replace('<script src="/remote.js"></script>','<script>'+ (ROOT/'static/remote.js').read_text()+'</script>')
 page.set_content(html,wait_until='domcontentloaded',timeout=15000)
 page.locator('#guide .channel-cell').first.wait_for(timeout=20000)
 assert page.locator('#remoteHelp').count()==1
 assert page.evaluate('typeof AussieTVRemote.keyOf')=='function'
 # Arrow with body initially focused should move to guide, not scroll
 page.locator('body').focus()
 page.keyboard.press('ArrowDown')
 assert page.evaluate('document.activeElement.matches("#guide .channel-cell")')
 page.keyboard.press('ArrowDown')
 assert page.evaluate('document.activeElement.dataset.ch')=='abc-news'
 page.keyboard.press('ArrowRight')
 assert page.evaluate('document.activeElement.matches(".show-cell")')
 page.keyboard.press('Enter')
 assert page.locator('#nowName').inner_text()=='ABC NEWS'
 # Remote Help open and return focus
 page.keyboard.press('h')
 assert page.locator('#remoteHelpPanel').is_visible()
 assert page.evaluate('document.activeElement.id')=='remoteHelpClose'
 page.keyboard.press('Escape')
 assert not page.locator('#remoteHelpPanel').is_visible()
 # Jump to browse filters via M and apply through Enter
 page.keyboard.press('m')
 assert page.evaluate('document.activeElement.closest("#filters") !== null')
 page.keyboard.press('ArrowDown')
 page.keyboard.press('Enter')
 assert page.locator('.side-item.active').count()==1
 # Text overlay uses focus-trap and Back close
 page.keyboard.press('t')
 assert page.locator('#textPanel').is_visible()
 assert page.evaluate('document.activeElement.dataset.textpage')=='100'
 page.keyboard.press('ArrowRight')
 page.keyboard.press('Escape')
 assert not page.locator('#textPanel').is_visible()
 # Numeric tuning with delay (not video playback), area selection
 page.keyboard.press('2');page.keyboard.press('4');page.wait_for_timeout(1300)
 assert page.locator('#nowName').inner_text()=='ABC NEWS'
 page.keyboard.press('g')
 assert page.evaluate('document.activeElement.matches("#guide .channel-cell")')
 page.locator('#remoteHelp').click()
 assert page.locator('#remoteHelpPanel').is_visible()
 page.locator('#remoteHelpClose').click()
 assert not page.locator('#remoteHelpPanel').is_visible()
 assert not errors, errors
 print('PASS: focus navigation, guide selection, sidebar, dialog Back, Text, numeric channel tuning, no page errors')
 browser.close()
server.shutdown()

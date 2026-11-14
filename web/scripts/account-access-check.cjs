// All writes are intercepted; no live account is created or reset.
const { chromium }=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const assert=require('node:assert/strict');
const fs=require('node:fs');
(async()=>{
 const browser=await chromium.launch({executablePath:process.env.CHROMIUM_PATH,args:['--no-sandbox']});
 try{
 const context=await browser.newContext({viewport:{width:1440,height:980}});
 const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
 let captchaCount=0,sent=0,registered=0,reset=0;
 const svg='<svg xmlns="http://www.w3.org/2000/svg" width="200" height="60"><rect width="200" height="60" fill="#e6f0ff"/><text x="25" y="43" font-size="39" letter-spacing="14" fill="#215cb5">1234</text></svg>';
 await context.route('**/api/v1/**',route=>{
  const path=new URL(route.request().url()).pathname;let data={};
  if(path.endsWith('/captcha')){captchaCount++;data={image:'data:image/svg+xml;base64,'+Buffer.from(svg).toString('base64'),idKey:'test-'+captchaCount};}
  else if(path.endsWith('/auth/options'))data={registrationEnabled:true,emailEnabled:true};
  else if(path.endsWith('/auth/email-code')){sent++;data={verificationId:'a'.repeat(64),retryAfter:60};}
  else if(path.endsWith('/auth/register')){registered++;assert.equal(route.request().postDataJSON().username,'test_operator');data={approvalRequired:true};}
  else if(path.endsWith('/auth/reset-password')){reset++;data={};}
  return route.fulfill({contentType:'application/json',body:JSON.stringify({code:200,data})});
 });
 await page.goto(process.env.BASE_URL||'http://127.0.0.1:8080/login');
 await page.locator('.auth-captcha img').waitFor();
 assert.equal(await page.getByLabel('账号',{exact:true}).inputValue(),'');
 assert.equal(await page.getByLabel('密码',{exact:true}).inputValue(),'');
 const fit=await page.locator('.auth-captcha img').evaluate(e=>getComputedStyle(e).objectFit);assert.equal(fit,'contain');
 await page.getByRole('button',{name:'刷新验证码',exact:true}).click();await page.waitForTimeout(120);assert.ok(captchaCount>=2);
 fs.mkdirSync('assets/screenshots',{recursive:true});
 await page.screenshot({path:'assets/screenshots/login-light.png',fullPage:true});
 await page.getByRole('button',{name:'切换到暗色主题',exact:true}).click();
 await page.waitForTimeout(200);assert.match(await page.locator('#app-favicon').getAttribute('href'),/mark-dark/);
 await page.screenshot({path:'assets/screenshots/login-dark.png',fullPage:true});
 await page.getByRole('tab',{name:'注册账号',exact:true}).click();
 await page.getByLabel('邮箱',{exact:true}).fill('test@example.com');
 await page.getByPlaceholder('4 位数字').fill('1234');
 await page.getByRole('button',{name:'发送邮箱验证码',exact:true}).click();
 await page.getByLabel('邮箱验证码',{exact:true}).waitFor();
 await page.getByLabel('账号',{exact:true}).fill('test_operator');
 await page.getByLabel('邮箱验证码',{exact:true}).fill('123456');
 await page.getByLabel('密码',{exact:true}).fill('test-password-123');
 await page.getByLabel('确认密码',{exact:true}).fill('mismatched-password');
 await page.getByRole('button',{name:'完成注册',exact:true}).click();assert.equal(registered,0);
 await page.getByLabel('确认密码',{exact:true}).fill('test-password-123');
 await page.getByRole('button',{name:'完成注册',exact:true}).click();
 await page.getByRole('status').filter({hasText:'注册成功'}).waitFor();assert.equal(registered,1);
 await page.getByRole('button',{name:'返回登录',exact:true}).click();
 await page.getByRole('button',{name:'忘记密码？',exact:true}).click();
 await page.getByLabel('邮箱',{exact:true}).fill('test@example.com');await page.getByPlaceholder('4 位数字').fill('1234');
 await page.getByRole('button',{name:'发送邮箱验证码',exact:true}).click();
 await page.getByLabel('邮箱验证码',{exact:true}).fill('123456');await page.getByLabel('新密码',{exact:true}).fill('new-password-456');await page.getByLabel('确认密码',{exact:true}).fill('new-password-456');
 await page.getByRole('button',{name:'重置密码',exact:true}).click();await page.getByRole('status').filter({hasText:'密码已重置'}).waitFor();assert.equal(reset,1);assert.equal(sent,2);
 await page.getByRole('button',{name:'返回登录',exact:true}).click();
 await page.getByRole('button',{name:'切换到亮色主题',exact:true}).click();
 await page.setViewportSize({width:390,height:844});await page.screenshot({path:'assets/screenshots/login-mobile.png',fullPage:true});
 assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
 await page.locator('.language-switch').click();await page.getByRole('menuitem',{name:'English',exact:true}).click();await page.getByRole('heading',{name:'Welcome back',exact:true}).waitFor();
 await page.emulateMedia({reducedMotion:'reduce'});assert.equal(await page.locator('.network-pulse').evaluate(e=>getComputedStyle(e).animationName),'none');
 assert.deepEqual(errors,[]);console.log('PASS: login defaults empty, complete captcha, light/dark branding, registration approval, password mismatch, recovery, English, mobile, reduced motion.');
 } finally {await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});

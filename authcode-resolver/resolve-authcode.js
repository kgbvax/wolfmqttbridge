const puppeteer = require('puppeteer')

const express = require('express')
const bodyParser = require('body-parser')


const app = express()
app.use(bodyParser.json()) // support json encoded bodies
app.use(bodyParser.urlencoded({ extended: true })) // support encoded bodies
const port = 3000

app.post('/', async (req, res) => {
  const username = req.body.username;
  const password = req.body.password;
  const browser = await puppeteer.launch();
  const page = await browser.newPage();
  await page.goto('https://www.wolf-smartset.com/');
  await page.type('input[name="Input.Username"]', username, {delay: 20})
  await page.type('input[name="Input.Password"]', password, {delay: 20})
  await page.keyboard.press('Enter')

  await page.waitForTimeout( 1000 );
  const sessionStorage = await page.evaluate(() =>  Object.assign({}, window.sessionStorage))
  const something = JSON.parse(sessionStorage["oidc.user:https://www.wolf-smartset.com/idsrv:smartset.web"])
  await browser.close();
  console.log(something)
  res.send(something)
})

app.listen(port, () => {
  console.log(`Authcode-resolver app listening at http://localhost:${port}`)
})



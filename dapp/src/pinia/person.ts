import { defineStore } from "pinia";
import { ETH } from "@/tools/contract";
import request from "@/tools/request";
import lang from '@/i18n/index'
import fetchSign from './fetchSign'
import { displayAmount } from '@/tools/amount'

const trimInfo = (obj: any): any => {
  if (!obj || typeof obj !== 'object') return obj
  if (Array.isArray(obj)) return obj.map(trimInfo)
  const out: any = { ...obj }
  for (const k of Object.keys(out)) {
    const v = out[k]
    if (typeof v === 'string' && /^-?\d+\.\d+$/.test(v)) out[k] = displayAmount(v)
    else if (v && typeof v === 'object') out[k] = trimInfo(v)
  }
  return out
}
import { showFailToast, showToast, showDialog, setToastDefaultOptions } from "vant";

setToastDefaultOptions({
  zIndex: 6001
})

let timeSwitch: any = null//定时获取用户信息
export default defineStore('person', {
  state: () => ({
    loadAccount: false,
    isLogin: false,
    userinfo: {
      status: 'ok',
      level: '0',
      locationNum: '0',
      total: '0',
      max: '0',
      min: '0',
      inviteUserAddress: '0x36fEa8A26AaD9Be34B29383D46FEaB42332389e6',
      buy: '0.00',
      amountGetSub: '0.00',
      amountGet: '0.00',
      outNum: '0',
      location: '0.00',
      recommend: '0.00',
      recommendTwo: '0.00',
      team: '0.00',
      teamTwo: '0.00',
      all: '0.00',
      usdt: '0.00',
      withdrawRate: 0.1,
      withdrawMin: 10,
      raw: '0.00',
      withdrawRateTwo: 0,
      withdrawMinTwo: 0,
      notice: '',
      goods: [],
      one: '', // 国家
      two: '', // 省份
      three: '', // 城市
      four: '', // 区域
      five: '', // 详情
      six: '', // 收件人手机
      seven: '', // 收件人
      shippingAddress: { name: '', contact: '', address: '' }
    },
    urlCode: '',
    sign: '',
    address: '',
    inviteUserAddress: '',
    isOpened: false,
    showCodeInput: false //输入邀请码弹窗
  }),
  actions: {
    // 系统初始化
    async init() {
      // await this.inputInvitationCode()
      // this.isLogin = true;
      // return
      let url = window.location.href

      console.log(url, /invitecode/.test(url.toLowerCase()), url.replace(/^(.*)(-invitetdh-)(.*)(-invitetdh-)(.*)$/gi, '$3'))
      if (/invitecode/.test(url.toLowerCase())) {
        this.urlCode = url.replace(/^(.*)(-invitetdh-)(.*)(-invitetdh-)(.*)$/gi, '$3')
      }

      const account = await ETH.getAccount()
      const accountLac = localStorage.getItem('account')

      this.address = account
      const login = async (params: any): Promise<string> => {
        let res: any = await request.post('app_server/eth_authorize', params)
        if (res.status === '用户已锁定') throw new Error(res.status)
        if (res.status === '无效的推荐码') throw new Error(res.status)
        if (res.status === '请输入推荐码') throw new Error(res.status)
        if (res.status !== 'ok' || !res.token) throw new Error(res.status || 'fail')
        return res.token
      }

      const sign = await fetchSign()
      this.sign = sign

      const checkLogin = async (sign: string) => {
        try {
          // 判断是否直接进入系统
          this.loginSuccess(await login({ address: ETH.account, code: '', sign, noMsg: true }))
        } catch (err: any) {
          if (err.message === '用户已锁定') {
            showFailToast(lang('用户已锁定'));
            return
          }
          // 根据推荐码进入系统
          try {
            const code = await this.inputInvitationCode()
            this.loginSuccess(await login({ address: ETH.account, code, sign, loading: true }))
          } catch (error: any) {
            showDialog({
              title: lang('错误'),
              message: lang('无效推荐码'),
            }).then(() => {
              checkLogin(sign)
            })
          }
          // this.showCodeInput = true;
        }
      }

      /* 判断是否本地token */
      if (accountLac === account) {
        this.loginSuccess()
      } else {
        localStorage.removeItem('token')
        checkLogin(sign)
      }
      this.loadAccount = true
    },
    async inputInvitationCode() {
      return new Promise((resolve, reject) => {
        const urlParams = new URLSearchParams(window.location.search)
        // 获取单个参数
        let code = urlParams.get('code')

        code = code === null || code === 'null' ? '' : code.trim()

        let inviteCode = ''

        showDialog({
          title: "Invite Code",
          message: `
            <input
              id="inviteInput"
              type="text"
              value="${this.urlCode}"
              placeholder="please enter the invitation code"
              style="
                width: calc(100% - 16px);
                padding:8px;
                border:1px solid #ddd;
                border-radius:6px;
              "
            />
          `,
          confirmButtonText: 'Confirm',
          allowHtml: true,
          zIndex: 6000,
          beforeClose(action: any) {
            if (action === 'confirm') {
              const value = (document.getElementById('inviteInput') as HTMLInputElement).value
              if (!value) {
                showToast({
                  message: "Invite Code",
                  type: 'fail'
                })
                return false // ❌ 阻止关闭
              }
              resolve(value)
            }
            return true
          }
        } as any)
        // .then(() => {
        //   console.log(3333)
        //   // const input = contentDiv.querySelector('.dialog-input') as HTMLInputElement
        //   // const inputCode = input ? input.value.trim() : ''
        //   // if (!inputCode) {
        //   //   showFailToast(lang('邀请码不能为空'))
        //   //   // 重新显示对话框
        //   //   this.inputInvitationCode().then(resolve).catch(reject)
        //   // } else {
        //   //   resolve(inputCode)
        //   // }
        // })
        // .catch(() => {
        //   // 用户取消，拒绝Promise
        //   reject(new Error('User canceled'))
        // })

        // 将内容添加到对话框中
        // setTimeout(() => {
        //   const dialog = document.querySelector('.van-dialog') as HTMLElement
        //   if (dialog) {
        //     const messageContainer = dialog.querySelector('.van-dialog__message') as HTMLElement
        //     if (messageContainer) {
        //       messageContainer.appendChild(contentDiv)
        //       // 自动聚焦输入框
        //       const input = contentDiv.querySelector('.dialog-input') as HTMLInputElement
        //       if (input) {
        //         input.focus()
        //       }
        //     }
        //   }
        // }, 100)
      })
    },
    async codeError(sign: string) {
      const code = await this.inputInvitationCode()

      const login = async (params: any): Promise<string> => {
        let res: any = await request.post('app_server/eth_authorize', params)
        if (res.status === '无效的推荐码') throw new Error(res.status)
        return res.token
      }

      try {
        this.loginSuccess(await login({ address: ETH.account, code, sign, loading: true }))
      } catch (error: any) {
        // f7.dialog.alert(error?.response?.data?.message || '无效的邀请码', '错误', async () => {
        //   await this.codeError(sign)
        // });
      }
    },
    /* 获取用户信息 */
    async getUser() {
      const getData = async () => {
        let res: any = trimInfo(await request.get('app_server/user_info'))
        const price = res.ispayPrice || res.priceTwo || '2000'
        const goods = res.goods || []
        const goodsThree = (res.goodsThree && res.goodsThree.length) ? res.goodsThree : goods.map((g: any) => ({
          id: g.id,
          one: g.title || g.goods,
          three: g.three || '',
          four: displayAmount(g.amount),
          amount: displayAmount(g.amount)
        }))
        this.userinfo = {
          ...this.userinfo,
          ...res,
          goodsThree,
          ispay: res.ispay || res.ispayAmount || '0',
          ispayAmount: res.ispayAmount || res.ispay || '0',
          rechargeBalance: res.rechargeBalance || res.amountUsdt || '0',
          amountUsdt: res.amountUsdt || res.rechargeBalance || '0',
          raw: res.ispay || res.ispayAmount || '0',
          rawNew: res.ispayAmount || res.ispay || '0.00',
          priceOne: res.priceOne || price,
          priceTwo: res.priceTwo || price,
          priceThree: res.priceThree || price,
          priceFour: res.priceFour || price,
          priceFive: res.priceFive || price,
          priceSix: res.priceSix || price,
          priceOneNew: res.priceOneNew || price,
          priceTwoNew: res.priceTwoNew || price,
          priceThreeNew: res.priceThreeNew || price,
          userAddress: res.userAddress || []
        }
      }
      clearInterval(timeSwitch)
      await getData()
      timeSwitch = setInterval(getData, 30000)
    },
    /* 登录成功 */
    async loginSuccess(token?: string) {
      if (token) {
        localStorage.setItem('token', token)
        localStorage.setItem('account', ETH.account)
      }
      await this.getUser()
      // await this.recommend_update();
      this.isLogin = true
      location.hash = ''
      this.urlCode = ''
    },
    /* 更新推荐关系 */
    async recommend_update() {
      const sign = await fetchSign()

      if (this.urlCode && this.urlCode !== this.inviteUserAddress) {
        let res: any = await request.post('app_server/recommend_update', {
          code: this.urlCode,
          sign
        })
        this.inviteUserAddress = res.inviteUserAddress
      }
    },
    /* 自动退出系统 */
    outLogin() {
      localStorage.removeItem('token')
      localStorage.removeItem('account')
      localStorage.removeItem('sign')
      this.$reset()
      this.init()
    }
  }
})
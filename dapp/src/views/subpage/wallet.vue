<template>
<div class='page'>
  <van-nav-bar
    :title="lang('我的资产')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
  />
  <div class="page-main">
    <div class="usdt-price" @click="router.push('/withdraw/usdt')">
      <div class="price-list">
        <div class="price-item">
          <p>USDT</p>
          <p>{{ userinfo.usdt || 0 }}<van-icon style="margin-left: 5px;" name="arrow" /></p>
        </div>
      </div>
    </div>
    <div class="ispay-price" @click="router.push('/withdraw/newIspay')">
      <p>ISPAY</p>
      <p>{{ userinfo.ispay || userinfo.ispayAmount || 0 }}<van-icon style="margin-left: 5px;" name="arrow" /></p>
    </div>
    <ul class="wallet-tab">
      <li :class="tab === 1 ? 'active' : ''" @click="tab = 1">{{ lang('我的收益') }}</li>
      <li :class="tab === 2 ? 'active' : ''" @click="tab = 2">{{ lang('双轨安置关系') }}</li>
      <li :class="tab === 3 ? 'active' : ''" @click="tab = 3">{{ lang('邀请关系') }}</li>
    </ul>
    <div class="tab-content" v-if="tab === 1">
      <div class="pledge">
        <div class="pledge-info">
          <div class="pledge-item">
            <p>{{ lang('待释放') }}ISPAY:</p>
            <p>{{ userinfo.amountGetSub || 0 }}</p>
          </div>
          <div class="pledge-item">
            <p>{{ lang('已释放') }}ISPAY:</p>
            <p>{{ userinfo.amountGet || 0 }}</p>
          </div>
        </div>
        <div class="pledge-count">
          {{ lang('释放次数') }}<span>{{ userinfo.outNum || 0 }}</span>
        </div>
      </div>
      <div class="pledge-frame">
        <div class="pledge-frame-item">
          <p>{{ lang('直推收益') }}</p>
          <p>{{ userinfo.recommend || 0 }}</p>
        </div>
        <div class="pledge-frame-item">
          <p>{{ lang('对碰收益') }}</p>
          <p>{{ userinfo.recommendTwo || 0 }}</p>
        </div>
        <div class="pledge-frame-item">
          <p>{{ lang('管理收益') }}</p>
          <p>{{ userinfo.team || 0 }}</p>
        </div>
        <div class="pledge-frame-item">
          <p>{{ lang('全部收益') }}</p>
          <p>{{ allIncome }}</p>
        </div>
      </div>
      <van-tabs v-model:active="active" @click-tab="onClickTab">
        <van-tab v-for="value in menuType" :title="value[1]" :name="value[0]">
          <van-empty v-if="listEmpty" :description="lang('暂无数据')" :image="emptyImage" />
          <div class="income-list" v-else>
            <div class="income-list-main">
              <div class="income-list-item" v-for="(item, index) in currentList" :key="item.id || index">
                <div class="income-list-item-info">
                  <template v-if="active === '1'">
                    <p>{{ item.title || item.goods || '-' }}</p>
                    <p>{{ lang('套餐内容') }}：{{ item.goods || item.title || '-' }}</p>
                    <p>{{ lang('套餐金额') }} {{ displayAmount(item.amount) }} USDT · {{ item.release_days || '-' }} {{ lang('天') }}</p>
                    <p>{{ lang('购买日期') }}：{{ item.purchase_date || item.paid_at || item.createdAt || item.created_at || '-' }}</p>
                  </template>
                  <template v-else>
                    <p>
                      <span>USDT {{ lang('数量') }}：{{ item.amount }}</span>
                    </p>
                    <p v-if="item.orderTitle || item.orderNo" style="font-size: 13px;">
                      {{ item.orderTitle || item.orderNo }}
                    </p>
                    <p v-if="item.detail" style="font-size: 13px;">{{ item.detail }}</p>
                    <p v-else-if="item.address" style="font-size: 13px;">{{ formatAddress(item.address) }}</p>
                    <p style="font-size: 12px;">{{ item.settleDate || item.createdAt }}</p>
                  </template>
                </div>
                <div v-if="active !== '1'" class="income-list-item-money">{{ item.reward || item.amount }}</div>
              </div>
              <Pagination
                v-if="active !== '1'"
                v-model="page"
                :page-count="allPageCount"
                mode="simple"
                @change="getRewardList"
              />
            </div>
          </div>
        </van-tab>
      </van-tabs>
    </div>
    <div class="tab-content" v-if="tab === 2">
      <div class="relation-card">
        <div class="tree-hint">{{ lang('点击下级查看下一层') }}</div>
        <div v-if="trail.length > 1" class="tree-trail">
          <span v-for="(item, idx) in trail" :key="item.user_id || idx">
            <a v-if="idx < trail.length - 1" @click="jumpTrail(idx)">{{ shortAddr(item.address) }}</a>
            <span v-else>{{ shortAddr(item.address) }}</span>
            <span v-if="idx < trail.length - 1"> / </span>
          </span>
        </div>
        <div v-if="placeRoot" class="org-scroll">
          <div class="org-tree">
            <org-branch :node="placeRoot" @select="enterDownline" />
          </div>
        </div>
        <van-empty v-else :description="lang('暂无数据')" :image="emptyImage" />
      </div>
    </div>
    <div class="tab-content" v-if="tab === 3">
      <div class="relation-card">
        <div class="tree-hint">{{ lang('点击下级查看下一层') }}</div>
        <div v-if="trail.length > 1" class="tree-trail">
          <span v-for="(item, idx) in trail" :key="'inv-' + (item.user_id || idx)">
            <a v-if="idx < trail.length - 1" @click="jumpTrail(idx)">{{ shortAddr(item.address) }}</a>
            <span v-else>{{ shortAddr(item.address) }}</span>
            <span v-if="idx < trail.length - 1"> / </span>
          </span>
        </div>
        <div v-if="inviteRoot" class="org-scroll">
          <div class="org-tree">
            <org-branch :node="inviteRoot" @select="enterDownline" />
          </div>
        </div>
        <van-empty v-else :description="lang('暂无数据')" :image="emptyImage" />
      </div>
    </div>
  </div>
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import { useRouter } from 'vue-router'
import emptyImage from '../../assets/images/custom-empty-image.png'
import request from "@/tools/request";
import { Pagination } from "vant"
import lang from '@/i18n/index'
import { displayAmount } from '@/tools/amount'
import OrgBranch from './components/OrgBranch.vue'

const router = useRouter()
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const allIncome = $computed(() => {
  const n = (v) => Number(String(v ?? 0).replace(/,/g, '')) || 0
  return displayAmount(
    n(userinfo.location) + n(userinfo.recommend) + n(userinfo.recommendTwo) + n(userinfo.team)
  )
})

const active = $ref('1')
const tab = $ref(1)
let page = $ref(1);
let allPageCount = $ref(1);
let rewardList = $ref([]);
let orderList = $ref([]);
let downline = $ref(null)
let trail = $ref([])

const menuType = [
  ['1', 'web3 ' + lang('认购')],
  ['2', lang('静态收益')],
  ['3', lang('直推收益')],
  ['4', lang('对碰奖励')],
  ['5', lang('管理奖励')],
]

const currentList = $computed(() => active === '1' ? orderList : rewardList)
const listEmpty = $computed(() => !currentList || currentList.length === 0)

const shortAddr = (addr) => {
  if (!addr) return '-'
  if (addr.length <= 14) return addr
  return addr.slice(0, 6) + '…' + addr.slice(-4)
}

const formatAddress = (value) => {
  if (!value) return ''
  const frontSix = value.slice(0, 6);
  const backSix = value.slice(-4);
  return frontSix + '...' + backSix;
}

const toInviteBranch = (it) => {
  return {
    key: 'inv-' + (it.user_id || it.address),
    person: it,
    display: shortAddr(it.address),
    fullAddress: it.address,
    meta: lang('已支付') + ' ' + (it.paid || '0'),
    cls: 'invite',
    children: [],
  }
}

const toPlaceBranch = (n, side, parentKey) => {
  const label = side === 'L' ? lang('左区') : lang('右区')
  if (!n) {
    return {
      key: 'empty-' + parentKey + '-' + side,
      empty: true,
      label,
      display: lang('空位'),
      fullAddress: '',
      meta: '',
      cls: 'empty ' + (side === 'L' ? 'left' : 'right'),
      children: [],
    }
  }
  return {
    key: 'pl-' + n.user_id + '-' + side,
    person: n,
    label,
    display: shortAddr(n.address),
    fullAddress: n.address,
    meta: (n.amount || n.paid || '0'),
    cls: side === 'L' ? 'left' : 'right',
    children: [],
  }
}

const inviteRoot = $computed(() => {
  const current = downline?.current
  if (!current || !current.address) return null
  return {
    key: 'inv-root-' + current.user_id,
    person: current,
    label: lang('当前'),
    display: shortAddr(current.address),
    fullAddress: current.address,
    meta: lang('已支付') + ' ' + (current.paid || '0'),
    cls: 'current',
    children: (downline.invites || []).map((it) => toInviteBranch(it)),
  }
})

const placeRoot = $computed(() => {
  const current = downline?.current
  if (!current || !current.address) return null
  return {
    key: 'pl-root-' + current.user_id,
    person: current,
    label: lang('当前'),
    display: shortAddr(current.address),
    fullAddress: current.address,
    meta: lang('已支付') + ' ' + (current.paid || '0'),
    cls: 'current',
    children: [
      toPlaceBranch(downline.left, 'L', String(current.user_id)),
      toPlaceBranch(downline.right, 'R', String(current.user_id)),
    ],
  }
})

const getOrderList = async () => {
  const res = await request.get("app_server/order_list")
  const rows = Array.isArray(res) ? res : (res.list || res.items || [])
  orderList = [...rows].sort((a, b) => {
    const ta = a.purchase_date || a.paid_at || a.createdAt || a.created_at || ''
    const tb = b.purchase_date || b.paid_at || b.createdAt || b.created_at || ''
    return String(tb).localeCompare(String(ta))
  })
}

const getRewardList = async (nextPage = 1) => {
  if (active === '1') {
    await getOrderList()
    return
  }
  const res = await request.get("app_server/reward_list", {
    params: {
      page: nextPage,
      reqType: active
    }
  });
  allPageCount = Math.max(1, Math.ceil((res.count || 0) / 10));
  rewardList = res.list || []
}

const getDownline = async (person, resetTrail) => {
  try {
    const params = {}
    if (person?.address) params.address = person.address
    downline = await request.get("app_server/downline", { params })
    const node = {
      user_id: downline?.current?.user_id,
      address: downline?.current?.address,
    }
    if (!node.address) return
    if (resetTrail || trail.length === 0) {
      trail = [node]
    } else {
      const last = trail[trail.length - 1]
      if (!last || String(last.user_id) !== String(node.user_id)) {
        trail = trail.concat([node])
      }
    }
  } catch (e) {
    downline = null
  }
}

const enterDownline = (person) => {
  if (!person || person.empty || !person.address) return
  if (downline?.current && String(downline.current.user_id) === String(person.user_id)) return
  getDownline(person, false)
}

const jumpTrail = (idx) => {
  const node = trail[idx]
  trail = trail.slice(0, idx)
  getDownline(node, false)
}

getRewardList()
getDownline(null, true)

const onClickTab = (tab) => {
  rewardList = []
  active = tab.name;
  page = 1
  getRewardList()
}

const handleBack = () => {
  router.back()
}

</script>
<style lang='less' scoped>
  .page {
    min-height: 100vh;
    box-sizing: border-box;
    padding: 50px 15px 20px 15px;
    background: url('../../assets/images/a3.png') no-repeat;
    background-size: 100% auto;
    .page-main {
      display: flex;
      flex-direction: column;
      gap: 20px;
    }
    .usdt-price {
      width: 100%;
      height: 111px;
      background-image: url(@/assets/images/a1.png);
      background-repeat: no-repeat;
      background-size: 100% 111px;
      padding: 32px 20px 0 20px;
      display: flex;
      box-sizing: border-box;
      font-size: 18px;
      font-size: 15px;
      .price-list {
        display: flex;
        gap: 10px;
        width: 100%;
        .price-item {
          display: flex;
          flex-direction: column;
          gap: 10px;
        }
      }
    }
    .wallet-tab {
      width: 100%;
      min-height: 45px;
      background: #23282c;
      border: 1px solid #33383f;
      border-radius: 31px;
      padding: 5px;
      box-sizing: border-box;
      display: flex;
      li {
        flex: 1 0 0;
        display: flex;
        align-items: center;
        justify-content: center;
        text-align: center;
        font-size: 13px;
        padding: 0 4px;
        &.active {
          height: 45px;
          background: #3640f0;
          border-radius: 26px;
          font-weight: 500;
        }
      }
    }
    .ispay-price {
      width: 100%;
      height: 111px;
      background-image: url(@/assets/images/a2.png);
      background-repeat: no-repeat;
      background-size: 100% 111px;
      padding: 25px 20px;
      box-sizing: border-box;
      display: flex;
      flex-direction: column;
      gap: 20px;
      font-size: 18px;
    }
    .tab-menu {
      display: flex;
      height: 55px;
      background: #23282c;
      border: 1px solid #33383f;
      border-radius: 31px;
      padding: 5px;
      box-sizing: border-box;
      .tab-item {
        flex: 1;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .active-tab {
        background: #3640f0;
        border-radius: 32px;
        font-weight: 500;
      }
    }
    .tab-content {
      display: flex;
      flex-direction: column;
      gap: 20px;
      .pledge {
        background: #23282c;
        border-radius: 18px;
        overflow: hidden;
        .pledge-total {
          display: flex;
          justify-content: space-between;
          margin: 15px;
          height: 57px;
          background: hsla(0, 0%, 100%, .1);
          border-radius: 12px;
          padding: 0 15px;
          box-sizing: border-box;
          align-items: center;
          span {
            &:nth-child(2) {
              color: rgb(255, 209, 39);
              font-weight: 500;
            }
          }
        }
        .pledge-info {
          display: flex;
          margin: 20px 15px;
          .pledge-item {
            flex: 1;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            gap: 6px;
            p {
              &:nth-child(2) {
                font-weight: 500;
                font-size: 18px;
              }
            }
          }
        }
        .pledge-count {
          border-top: 1px solid rgba(255, 255, 255, 0.1);
          background: url('@/assets/images/xian.png') no-repeat;
          background-size: 100% auto;
          padding: 60px 0 20px 0;
          text-align: center;
          span {
            color: rgb(255, 209, 39);
            font-weight: 500;
            margin-left: 10px;
          }
        }
      }
      .pledge-frame {
        height: 220px;
        background: url('@/assets/images/boxbg1.png') no-repeat;
        background-size: 100% 220px;
        box-sizing: border-box;
        display: flex;
        flex-wrap: wrap;
        .pledge-frame-item {
          width: 50%;
          height: 110px;
          display: flex;
          align-items: center;
          justify-content: center;
          flex-direction: column;
          gap: 10px;
          p {
            &:nth-child(2) {
              font-size: 18px;
              font-weight: 500;
            }
          }
        }
      }
      .pledge-earnings {
        min-height: 88px;
        background: #23282c;
        border-radius: 18px;
        display: flex;
        .pledge-earnings-item {
          height: 88px;
          flex: 1;
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: 6px;
          p {
            &:nth-child(2) {
              font-size: 16px;
              font-weight: 500;
            }
          }
        }
      }
      .pledge-give {
        height: 58px;
        background-image: url(@/assets/images/btnbg.png);
        background-repeat: no-repeat;
        background-size: 100% 58px;
        display: flex;
        align-items: center;
        justify-content: space-between;
        box-sizing: border-box;
        padding: 0 20px;
        span {
          &:nth-child(2) {
            font-size: 16px;
            font-weight: 500;
          }
        }
      }
      /deep/ .van-tab__panel {
        padding: 20px 0;
      }
    }
    .relation-row {
      display: flex;
      gap: 12px;
      align-items: stretch;
      overflow-x: auto;
    }
    .relation-card {
      flex: 1 1 50%;
      min-width: 240px;
      background: #23282c;
      border-radius: 18px;
      padding: 12px 8px 16px;
      box-sizing: border-box;
    }
    .relation-title {
      text-align: center;
      font-weight: 500;
      margin-bottom: 10px;
    }
    .tree-hint {
      text-align: center;
      font-size: 12px;
      color: #8a9199;
      margin-bottom: 8px;
    }
    .tree-trail {
      text-align: center;
      font-size: 12px;
      color: #c5ccd3;
      margin-bottom: 10px;
      word-break: break-all;
      a {
        color: #6ea8ff;
      }
    }
    .org-scroll {
      overflow-x: auto;
      padding: 4px 0 8px;
    }
    .org-tree {
      display: inline-block;
      min-width: 100%;
      text-align: center;
    }
    .section-title {
      text-align: center;
      font-weight: 500;
    }
    .income-box {
      display: flex;
      flex-direction: column;
      .income-main {
        display: flex;
        flex-direction: column;
        justify-content: center;
        gap: 20px;
        align-items: center;
        position: relative;
        padding-bottom: 20px;
        &::after {
          content: "";
          position: absolute;
          z-index: 1;
          bottom: 0;
          left: 0;
          width: 100%;
          height: 0.02564rem;
          background: linear-gradient(90deg,rgba(179,179,179,0) 0%,rgba(255,255,255,.6) 50.45%,rgba(179,179,179,0) 100%);
        }
        p {
          &:nth-child(1) {
            font-size: 14px;
            color: #CCC;
          }
          &:nth-child(2) {
            font-size: 26px;
            color: #FFF;
          }
        }
      }
      .income-footer {
        display: flex;
        flex-wrap: wrap;
        padding-top: 20px;
        gap: 20px 0;
        .income-footer-item {
          width: 25%;
          flex-grow: 1;
          flex-shrink: 0;
          align-items: center;
          display: flex;
          flex-direction: column;
          justify-content: flex-start;
          align-items: center;
          gap: 5px;
          p {
            &:nth-child(1) {
              font-size: 12px;
              color: #CCC;
            }
            &:nth-child(2) {
              font-size: 12px;
              color: #FFF;
              display: flex;
              gap: 4px;
              align-items: center;
            }
          }
        }
      }
    }
    .income-list {
      overflow: hidden;
      .list-menu-select {
        width: 100%;
        height: 40px;
        background: url('data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADAAAAAwCAQAAAD9CzEMAAAAtUlEQVR42u2VUQqCQBRF3y4FoUAoCooCoUAQnJ21gfl0PzdIbOhS+UbfT/HO53g9B0QYcRzHcX4SBPSICJIJOuV7uGGgzdK3GOinpxEjjVrfYCRqPlHiqtJfkAgieYl6cl0j0QmhSZy/Lk+sn5M4flwdWD83sX+72LF+SWIrBDasX5qoXp5UrLdIrJ+nK9ZbJcrHScn/vWWiQMF64wTrTRP2ek6w3j7BevtERETwa9lxHOfvuAOAC4GPzKVVpAAAAABJRU5ErkJggg==') no-repeat right center;
        background-size: 18px 18px;
      }
      .income-list-header {
        width: 100%;
        height: 40px;
        line-height: 40px;
        overflow-x: auto;
        padding-bottom: 10px;
        &::-webkit-scrollbar {
          height: 0;
        }
        .header-list {
          height: 40px;
          line-height: 40px;
          padding: 0 5px;
          display: flex;
          gap: 10px;
          li {
            display: flex;
            align-items: center;
            white-space: nowrap;
            padding: 0 15px;
            border-radius: 6px;
            &.active {
              background: #29313C;
            }
          }
        }
      }
      .income-list-main {
        display: flex;
        flex-direction: column;
        background: #29313C;
        padding: 10px;
        .income-list-item {
          width: 100%;
          box-sizing:border-box;
          -moz-box-sizing:border-box;
          -webkit-box-sizing:border-box;
          padding: 10px;
          border-bottom: 1px solid #242738;
          display: flex;
          justify-content: space-between;
          align-items: flex-start;
          gap: 8px;
          &:nth-child(2n) {
            background: #29313C;
          }
          .income-list-item-info {
            width: 100%;
            flex-grow: 1;
            display: flex;
            flex-direction: column;
            gap: 4px;
            p {
              width: 100%;
              color: #CCC;
              display: flex;
              flex-grow: 1;
              justify-content: space-between;
              &:nth-child(2) {
                font-size: 12px;
              }
            }
          }
          .income-list-item-money {
            flex-shrink: 0;
            width: 100px;
            text-align: right;
            color: #CCC;
            font-size: 15px;
            font-weight: 500;
            padding: 0 10px;
          }
        }
      }
    }
  }
</style>
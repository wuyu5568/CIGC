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
          <p>USDT<br />Bep20</p>
          <p>{{ userinfo.usdt || 0 }}<van-icon style="margin-left: 5px;" name="arrow" /></p>
        </div>
        <div class="price-item">
          <p>USD<br />Brc20</p>
          <p>{{ userinfo.rawNew || 0 }}<van-icon style="margin-left: 5px;" name="arrow" /></p>
        </div>
      </div>
    </div>
    <div class="ispay-price" @click="router.push('/withdraw/ispay')">
      <p>ISPAY</p>
      <p>{{ userinfo.raw || 0 }}<van-icon style="margin-left: 5px;" name="arrow" /></p>
    </div>
    <div class="ispay-price" @click="router.push('/withdraw/newIspay')">
      <p>ISPAY</p>
      <p>{{ userinfo.ispayAmount }}<van-icon style="margin-left: 5px;" name="arrow" /></p>
    </div>
    <ul class="wallet-tab">
      <li :class="tab === 1 ? 'active' : ''" @click="tab = 1">{{ lang('我的收益') }}</li>
      <li :class="tab === 2 ? 'active' : ''" @click="tab = 2">{{ lang('矩阵图') }}</li>
    </ul>
    <div class="tab-content" v-if="tab === 1">
      <div class="pledge">
        <div class="pledge-info">
          <div class="pledge-item">
            <p>{{ lang('待产出') }}</p>
            <p>{{ userinfo.amountGetSub || 0 }}</p>
          </div>
          <div class="pledge-item">
            <p>{{ lang('已产量') }}</p>
            <p>{{ userinfo.amountGet || 0 }}</p>
          </div>
        </div>
        <div class="pledge-count">
          {{ lang('出局次数') }}<span>{{ userinfo.outNum || 0 }}</span>
        </div>
      </div>
      <div class="pledge-frame">
        <div class="pledge-frame-item">
          <p>{{ lang('静态收益') }}</p>
          <p>{{ userinfo.location || 0 }}</p>
        </div>
        <div class="pledge-frame-item">
          <p>{{ lang('直推收益') }}</p>
          <p>{{ userinfo.recommend || 0 }}</p>
        </div>
        <div class="pledge-frame-item">
          <p>{{ lang('直推加速') }}</p>
          <p>{{ userinfo.recommendTwo || 0 }}</p>
        </div>
        <div class="pledge-frame-item">
          <p>{{ lang('团队收益') }}</p>
          <p>{{ userinfo.team || 0 }}</p>
        </div>
      </div>
      <div class="pledge-earnings">
        <div class="pledge-earnings-item">
          <p>{{ lang('平级收益') }}</p>
          <p>{{ userinfo.teamTwo || 0 }}</p>
        </div>
        <div class="pledge-earnings-item">
          <p>{{ lang('全网分红') }}</p>
          <p>{{ userinfo.all || 0 }}</p>
        </div>
      </div>
      <!-- <div class="pledge-give">
        <span>{{ lang('赠送 ISPAY') }}</span>
        <span>0</span>
      </div> -->
      <van-tabs v-model:active="active" @click-tab="onClickTab">
        <van-tab v-for="value in menuType" :title="value[1]" :name="value[0]">
          <van-empty v-if="rewardList.length === 0" :description="lang('暂无数据')" :image="emptyImage" />
          <div class="income-list" v-else>
            <div class="income-list-main">
              <div class="income-list-item" v-for="(item, index) in rewardList" :key="index">
                <div class="income-list-item-info">
                  <p>
                    <span v-if="!['8'].includes(active)">usdt {{ lang('数量') }}：{{ item.amount }}</span>
                    <span v-if="!['12', '13', '14'].includes(active)">{{item.name}} {{ lang('数量') }}：{{ item.amountTwo }}</span>
                  </p>
                  <p style="font-size: 13px;">
                    <span v-if="!['1', '2', '7', '8', '9', '10', '12', '14'].includes(active)">{{ formatAddress(item.address) }}</span>
                    <span v-if="!['1', '2', '9', '10', '3', '7', '8', '12', '14'].includes(active)">{{lang('代数')}}：{{ item.num }}</span>
                  </p>
                  <p style="font-size: 12px;">{{ item.createdAt }}</p>
                </div>
                <div class="income-list-item-money">{{ item.reward }}</div>
              </div>
              <Pagination
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
      <a-tree
        v-if="treeData.length > 0"
        v-model:expandedKeys="expandedKeys"
        v-model:selectedKeys="selectedKeys"
        :load-data="onLoadData"
        :tree-data="treeData"
      />
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

const router = useRouter()
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const address = $computed(() => person.address);

const active = $ref('1')
let page = $ref(1);
let allPageCount = $ref(3);
let rewardList = $ref([]);
const tab = $ref(1)

const menuType = [
  ['1', lang('认购')],
  ['9', lang('医疗商品认购')],
  ['10', 'web3 ' + lang('认购')],
  ['2', lang('静态收益')],
  ['3', lang('直推收益')],
  ['4', lang('直推加速收益')],
  ['5', lang('团代收益')],
  ['6', lang('平级收益')],
  ['7', lang('全网收益')],
  ['8', lang('赠送ispay')],
  ['12', lang('新预售2认购')],
  ['13', lang('新预售2购推荐收益')],
  ['14', lang('每日ispay收益')],
]
const expandedKeys = $ref([]);
const selectedKeys = $ref([]);
let treeData = $ref([]);

const getRewardList = async (page = 1) => {
  const res = await request.get("app_server/reward_list", {
    params: {
      page,
      reqType: active
    }
  });

  allPageCount = Math.ceil(res.count / 10);
  rewardList = res.list
}

getRewardList()

const formatAddress = (value) => {
  const frontSix = value.slice(0, 6);
  const backSix = value.slice(-4);
  const middle = '...';
  return frontSix + middle + backSix;
}

const collectKeys = (nodes, out = []) => {
  (nodes || []).forEach((n) => {
    out.push(n.key)
    if (n.children && n.children.length) collectKeys(n.children, out)
  })
  return out
}

const mapRecommend = (item, key, full) => {
  const kids = (item.children || []).map((c, i) => mapRecommend(c, `${key}-${i}`, full))
  return {
    title: `${formatAddress(item.address)}---(${lang('数量')}:${item.amount})`,
    key,
    amount: item.amount,
    address: item.address,
    isLeaf: kids.length === 0 && !!full,
    children: kids.length ? kids : (full ? [] : undefined)
  }
}

const onLoadData = (treeNode) => {
  return new Promise(async (resolve) => {
    if (treeNode.dataRef.children) {
      resolve();
      return;
    }

    const res = await request.get(`app_server/recommend_list?address=${treeNode.dataRef.address}`);

    // res.area = [
    //   {
    //     address: formatAddress("aaaa12378123hjdfowis88883"),
    //     locationId: "3",
    //     countLow: 2,
    //   },
    //   {
    //     address: formatAddress("32423998f8uijkjkrejkw2223"),
    //     locationId: "4",
    //     countLow: 0
    //   }
    // ]
    setTimeout(() => {
      const full = !!res.full
      treeNode.dataRef.children = (res.recommends || []).map((item, index) => mapRecommend(item, `${treeNode.eventKey}-${index}`, full))
      treeData = [...treeData];
      resolve();
    }, 1000);
  });
};

const getUserArea = async () => {
  const res = await request.get(`app_server/recommend_list?address=${address}`);
  const full = !!res.full
  treeData = (res.recommends || []).map((item, index) => mapRecommend(item, index, full))
  if (full) {
    expandedKeys = collectKeys(treeData)
  }
}

getUserArea()

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
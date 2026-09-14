<template>
<div class='withdraw-page'>
  <van-nav-bar
    :title="lang('充值')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
  />
  <div class="page-main">
    <div class="withdraw-info">
      <p class="withdraw-balance">USDT: {{ displayAmount(userinfo.amountUsdt) }}</p>
      <p class="withdraw-balance">BRC20: {{ userinfo.rawNew || 0.0000 }}</p>
      <button class="withdraw-btn" @click="showWithdraw"><van-icon name="balance-pay" />{{ lang('充值') }}</button>
    </div>
    <div class="withdraw-tab">
      <ul class="withdraw-tab-title">
        <li class="active">{{ lang('充值记录') }}</li>
      </ul>
      <div class="withdrawal-list">
        <div class="withdrawal-list-content">
          <div class="table">
            <div class="table-header" v-if="amountList.length > 0">
              <div class="table-row">
                <div class="table-cell">{{lang('日期')}}</div>
                <div class="table-cell">{{lang('金额')}}</div>
              </div>
            </div>
            <div class="table-body">
              <div class="table-row" v-for="(item, index) in amountList" :key="index">
                <div class="table-cell">{{item.createdAt}}</div>
                <div class="table-cell">{{ displayAmount(item.amount) }}</div>
              </div>
            </div>
          </div>
          <div class="empty" v-if="amountList.length === 0">
            <img :src="emptyImage" />
            <div class="empty-text">{{ lang('暂无数据') }}</div>
          </div>
          <Pagination
            v-if="amountList.length > 0"
            v-model="page"
            :page-count="allPageCount"
            mode="simple"
            @change="getAmountList"
          />
        </div>
      </div>
    </div>
  </div>
  <RechargeDialog :getBalance="getBalance" :usdtBalance="usdtBalance" :onChange="getAmountList" ref="rechargeDialogRef" />
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import { useRouter, useRoute } from 'vue-router'
import { Contract, ETH } from "@/tools/contract";
import { ref, computed } from 'vue'
import request from "@/tools/request";
import { showLoadingToast, closeToast, showFailToast, showDialog, closeDialog, showSuccessToast } from "vant";
import RechargeDialog from "./components/rechargeDialog.vue";
import emptyImage from '../../assets/images/custom-empty-image.png'
import { Pagination } from "vant"
import lang from '@/i18n/index'
import { displayAmount } from '@/tools/amount'

const USDT = new Contract(import.meta.env.VITE_USDT, "ERC20");
const BUY = new Contract(import.meta.env.VITE_BUY, "BUY");
const route = useRoute()

const router = useRouter()
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const rechargeDialogRef = ref(null)
const amountList = $ref([])
const page = $ref(1)
const allPageCount = $ref(1)
const usdtBalance = $ref("0");
const usdtApproved = $ref(false);

/* 获取授权 */
const getUsdtApproved = async () => {
    let res = await USDT.call("allowance", [ETH.account, BUY.address]);
    console.log('getUsdtApproved', Number(res))
    usdtApproved = Number(res) > 0;
    closeToast()

    return usdtApproved
}

const usdtApprove = async () => {
    showLoadingToast({
        message: lang('授权中'), duration: 0, overlay: true, overlayStyle: {
            background: "transparent"
        }
    });
    await USDT.send("approve", [
        BUY.address,
        "115792089237316195423570985008687907853269984665640564039457584007913129639935"
    ]).then(getUsdtApproved).catch(() => closeToast());
}

/* 获取代币余额 */
const getBalance = async () => {
  await ETH.getAccount()
  const res = await ETH.getUSDTBalance()
  usdtBalance = res;
}

const getData = async () => {
    // showLoadingToast({
    //     message: lang('授权中'), duration: 0, overlay: true, overlayStyle: {
    //         background: "transparent"
    //     }
    // });
    await getBalance();
    await getUsdtApproved();
    // closeToast();
}

getData()

function showWithdraw(type) {
  if (usdtApproved) {
    rechargeDialogRef.value?.open(type)
  } else {
    usdtApprove()
  }
}

const handleBack = () => {
  router.back()
}

const getAmountList = async (page = 1) => {
    await request.get("app_server/deposit_list", {
      params: {
        page
      }
    }).then((res) => {
      amountList = res.list || []
      allPageCount = Math.max(1, Math.ceil(Number(res.count || 0) / 10))
    })
}

getAmountList()

</script>
<style lang='less' scoped>
  .page-main {
    padding: 56px 15px 0 15px;
    .withdraw-info {
      width: 100%;
      aspect-ratio: 694 / 310;
      background: url('@/assets/images/boxbg3.png') no-repeat center;
      background-size: cover;
      box-sizing: border-box;
      padding: 20px;
      display: flex;
      margin-bottom: 20px;
      flex-direction: column;
      gap: 10px;
      align-items: flex-start;
      .withdraw-balance {
        font-size: 16px;
        color: #fff;
      }
      .withdraw-name {
        font-size: 16px;
        color: #fff;
      }
      .withdraw-btn {
        padding: 0 20px;
        display: inline-flex;
        height: 36px;
        margin-top: 10px;
        background: hsla(0, 0%, 100%, .1);
        border: 1px solid #999;
        border-radius: 22px;
        display: flex;
        gap: 5px;
        align-items: center;
        justify-content: center;
        font-size: 16px;
        i {
          font-size: 20px;
        }
      }
    }
    .withdraw-tab {
      .withdraw-tab-title {
        height: 44px;
        display: flex;
        gap: 20px;
        margin-top: 20px;
        border-top-left-radius: 20px;
        border-top-right-radius: 20px;
        background: linear-gradient(0deg,rgba(35,40,44,0),#252b4b);
        display: flex;
        align-items: center;
        justify-content: flex-start;
        padding: 0 20px;
        li {
          font-size: 14px;
          color: #FFF;
          position: relative;
          &.active {
            &::before {
              width: 25px;
              height: 2px;
              background: 0% 0% / cover rgb(54, 64, 240);
              content: '';
              display: block;
              position: absolute;
              bottom: -10px;
              left: 50%;
              transform: translateX(-50%);
              border-radius: 10px;
            }
          }
        }
      }
    }
    .empty {
      text-align: center;
      margin: 50px 0;
      img {
        width: 50px;
        height: 50px;
        margin: 0 auto;
        opacity: 0.3;
      }
      .empty-text {
        margin-top: 15px;
        font-size: 14px;
        color: rgba(255, 255, 255, 0.3);
      }
    }
    .withdraw-tab-content {
      padding: 50px 0;
    }
    .withdrawal-list {
      padding: 20px;
      .withdrawal-list-title {
        font-size: 16px;
      }
      .withdrawal-list-content {
        min-height: 200px;
        .table {
          display: table;
          width: 100%;
          border-collapse: collapse;
          .table-header {
            display: table-header-group;
            font-weight: bold;
            border-bottom: 0.5px dashed #999;
            .table-row {
              width: 100%;
              display: table-row;
            }
            .table-cell {
              display: table-cell;
              padding: 10px;
              text-align: left;
              box-sizing: border-box;
              font-weight: 500;
              color: #999;
              &:nth-child(2) {
                text-align: center;
              }
              &:nth-child(3) {
                text-align: center;
              }
            }
          }
          .table-body {
            display: table-row-group;
            overflow-y: auto;
            .table-row {
              width: 100%;
              display: table-row;
              border-bottom: 1px solid #444;
              &:last-child {
                border-bottom: none;
              }
            }
            .table-cell {
              display: table-cell;
              text-align: left;
              box-sizing: border-box;
              padding: 10px;
              p {
                height: 16px;
                padding: 5px 0;
                margin: 0;
              }
              &:nth-child(2) {
                text-align: center;
              }
              &:nth-child(3) {
                text-align: center;
              }
            }
          }
        }
      }
    }
    .recharge {
      min-height: 200px;
      background: rgba(23, 28, 33, .8);
      border: 1px solid #666;
      border-radius: 18px;
      padding: 15px;
      .recharge-info {
        .recharge-info-list {
          li {
            height: 32px;
            line-height: 32px;
            border-bottom: 1px solid #EEE;
            &::last-child {
              border-bottom: 0;
            }
          }
        }
      }
    }
  }
</style>

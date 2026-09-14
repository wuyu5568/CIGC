<template>
<div class='withdraw-page'>
  <van-nav-bar
    :title="lang('提现')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
  />
  <div class="page-main">
    <div class="withdraw-info">
      <p class="withdraw-balance">{{ route.params.type === 'usdt' ? userinfo.usdt : route.params.type === 'newIspay' ? userinfo.ispayAmount : userinfo.raw }}</p>
      <p class="withdraw-name">{{ route.params.type === 'usdt' ? 'USDT' : 'ISPAY' }}</p>
      <button class="withdraw-btn" @click="showWithdraw(route.params.type === 'usdt' ? 'USDT' : route.params.type === 'newIspay' ? 'NEWISPAY' : 'ISPAY')"><van-icon name="balance-pay" />{{ lang('提现') }}</button>
    </div>
    <div class="withdraw-tab">
      <ul class="withdraw-tab-title">
        <!-- <li :class="{active: active === '1'}" @click="active = '1'">账户记录</li> -->
        <li :class="{active: active === '2'}" @click="active = '2'">{{ lang('提现记录') }}</li>
        <li v-if="route.params.type === 'ispay'" :class="{active: active === '4'}" @click="active = '4'">{{ lang('锁仓记录') }}</li>
      </ul>
      <!-- <div class="withdraw-tab-content">
        <div class="empty">
          <img :src="emptyImage" />
          <div class="empty-text">{{ lang('暂无数据') }}</div>
        </div>
      </div> -->
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
                <div class="table-cell">{{item.amount}}</div>
              </div>
            </div>
          </div>
          <div class="empty" v-if="amountList.length === 0">
            <img :src="emptyImage" />
            <div class="empty-text">{{ lang('暂无数据') }}</div>
          </div>
          <Pagination
            v-model="allPage"
            :page-count="allPageCount"
            mode="simple"
            @change="getAmountList"
          />
        </div>
      </div>
    </div>
  </div>
  <WithdrawDialog ref="withdrawDialogRef" :onChange="updateList" />
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import { useRouter, useRoute } from 'vue-router'
import { ref, computed } from 'vue'
import request from "@/tools/request";
import { Pagination } from "vant"
import WithdrawDialog from "./components/withdrawDialog.vue";
import emptyImage from '../../assets/images/custom-empty-image.png'
import lang from '@/i18n/index'

const route = useRoute()

const router = useRouter()
const person = userPerson();
const active = $ref('2')
const userinfo = $computed(() => person.userinfo);
const withdrawDialogRef = ref(null)
const amountList = $ref([])
const allPage = $ref(1)
const allPageCount = $ref(1)

function showWithdraw(type) {
  withdrawDialogRef.value?.open(type)
}

const handleBack = () => {
  router.back()
}

const updateList = () => {
  route.params.type === 'usdt' && getAmountList()
}

const getAmountList = async (page = 1) => {
    await request.get("app_server/withdraw_list", {
      params: {
        coinType: route.params.type === 'newIspay' ? 3 : undefined,
        page
      }
    }).then((res) => {
      allPageCount = Math.ceil(res.count / 10);
      amountList = res.list
    })
}

(route.params.type === 'usdt' || route.params.type === 'newIspay') && getAmountList()

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
      flex-direction: column;
      gap: 10px;
      align-items: flex-start;
      .withdraw-balance {
        font-size: 20px;
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
  }
</style>
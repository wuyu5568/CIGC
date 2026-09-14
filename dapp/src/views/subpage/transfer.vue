<template>
<div class='transfer-page'>
  <van-nav-bar
    :title="lang('转账')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
  />
  <div class="page-main">
    <div class="transfer-info">
      <p>Brc20 {{lang('余额')}}：{{ userinfo.rawNew || 0.00 }}</p>
      <div class="transfer-item">
        <p>{{ lang('接收地址') }}</p>
        <input type="text" v-model="address" class="form-input" :placeholder="lang('请输入接收地址')" />
      </div>
      <div class="transfer-item">
        <p>{{ lang('转账金额') }}</p>
        <input type="number" v-model="amount" class="form-input" :placeholder="lang('请输入转账金额')" />
      </div>
      <button class="form-submit-btn" @click="handleSubmit">{{ lang('确认转账') }}</button>
    </div>
    <div class="withdraw-tab">
      <ul class="withdraw-tab-title">
        <li class="active">{{ lang('转账记录') }}</li>
      </ul>
      <div class="withdrawal-list">
        <div class="withdrawal-list-content">
          <div class="table">
            <!-- <div class="table-header" v-if="amountList.length > 0">
              <div class="table-row">
                <div class="table-cell">{{lang('日期')}}</div>
                <div class="table-cell">{{lang('金额')}}</div>
              </div>
            </div> -->
            <div class="table-body">
              <div class="table-row" v-for="(item, index) in amountList" :key="index">
                <div class="table-row-p">
                  <div class="table-cell">{{formatAddress(item.address)}}</div>
                  <div class="table-cell">{{item.createdAt}}</div>
                </div>
                <div class="table-row-p">
                  <div class="table-cell">{{item.amount}}</div>
                </div>
              </div>
            </div>
          </div>
          <div class="empty" v-if="amountList.length === 0">
            <img :src="emptyImage" />
            <div class="empty-text">{{ lang('暂无数据') }}</div>
          </div>
          <Pagination
            v-model="page"
            :page-count="allPageCount"
            mode="simple"
            @change="getRewardList"
          />
        </div>
      </div>
    </div>
  </div>
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import { useRouter, useRoute } from 'vue-router'
import request from "@/tools/request";
import { showLoadingToast, closeToast, showFailToast, showDialog, closeDialog, showSuccessToast } from "vant";
import { Pagination } from "vant"
import emptyImage from '../../assets/images/custom-empty-image.png'
import lang from '@/i18n/index'

const route = useRoute()

const router = useRouter()
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const sign = $computed(() => person.sign);
const amountList = $ref([])
const page = $ref(1);
const allPageCount = $ref(3);
const address = $ref(null)
const amount = $ref(null)

const formatAddress = (value) => {
  const frontSix = value.slice(0, 6);
  const backSix = value.slice(-4);
  const middle = '...';
  return frontSix + middle + backSix;
}

const getRewardList = async (page = 1) => {
  const res = await request.get("app_server/reward_list", {
    params: {
      page,
      reqType: 11
    }
  });

  allPageCount = Math.ceil(res.count / 10);
  amountList = res.list
}

getRewardList()

const handleSubmit = async () => {
  if (!address) {
    return showFailToast(lang('请输入接收地址'))
  }
  if (!amount) {
    return showFailToast(lang('请输入转账金额'))
  }
  if (Number(amount) > Number(userinfo.rawNew)) {
    return showFailToast(lang('余额不足'))
  }

  showLoadingToast({
      message: lang('转账中'), duration: 0, overlay: true, overlayStyle: {
          background: "transparent"
      }
  });

  await request.post("app_server/amount_to", {
    amount,
    address,
    sign
  }).then((res) => {
    if (res.status === 'ok') {
      address = null
      amount = null
      person.getUser();
      getRewardList()
      return showSuccessToast(lang('转账成功'))
    }
    showFailToast(res.status)
  })
}


const handleBack = () => {
  router.back()
}

</script>
<style lang='less' scoped>
  .page-main {
    padding: 56px 15px 0 15px;
    .transfer-info {
      min-height: 238px;
      background: rgba(23, 28, 33, .8);
      border: 1px solid #666;
      border-radius: 18px;
      padding: 15px;
      display: flex;
      flex-direction: column;
      gap: 20px;
      .transfer-item {
        display: flex;
        flex-direction: column;
        gap: 10px;
        p {
          font-size: 14px;
          color: #999;
        }
        .form-input {
          height: 41px;
          background: #23282c;
          border: 1px solid #666;
          border-radius: 8px;
          padding: 10px;
          box-sizing: border-box;
        }
      }
      .form-submit-btn {
        height: 42px;
        background: #1668dc;
        border: 0;
        border-radius: 8px;
        margin-top: 20px;
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
            font-weight: bold;
            border-bottom: 0.5px dashed #999;
            .table-row {
              width: 100%;
              display: flex;
            }
            .table-row-p {
              display: flex;
              flex-direction: column;
            }
            .table-cell {
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
            overflow-y: auto;
            .table-row {
              width: 100%;
              border-bottom: 1px solid #444;
              display: flex;
              align-items: center;
              padding-bottom: 10px;
              margin-bottom: 10px;
              &:last-child {
                border-bottom: none;
              }
            }
            .table-row-p {
              display: flex;
              flex-direction: column;
              &:nth-child(1) {
                flex: 1 0 0;
                display: flex;
                flex-direction: column;
                gap: 5px;
              }
              &:nth-child(2) {
                flex-shrink: 0;
              }
            }
            .table-cell {
              box-sizing: border-box;
              padding: 2px;
              p {
                height: 14px;
                padding: 0 0;
                margin: 0;
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
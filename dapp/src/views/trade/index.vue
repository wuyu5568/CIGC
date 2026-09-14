<template>
  <f7-page name="recharge">
    <ChildrenHeader />
    <div class='trade'>
      <div class="trade-title">{{ lang('交易') }}</div>
      <div class="trade-table">
        <ul>
          <li>
            <p>{{ lang('名称') }}</p>
            <p>ISPAY</p>
          </li>
          <li>
            <p>{{ lang('涨跌') }}</p>
            <p>{{userinfo.priceChange || 0}}%</p>
          </li>
          <li>
            <p>{{ lang('价格') }}</p>
            <p>{{ userinfo.price || 0}}</p>
          </li>
        </ul>
      </div>
      <div class="trade-form">
        <div class="trade-form-item">
          <div class="trade-form-label">
            <span>{{ lang('发送') }}</span>
            <span>{{ lang('余额') }}: {{ userinfo.balanceBiw }}</span>
          </div>
          <div class="trader-form-input">
            <div class="trader-form-input-name"><i class="biw-icon" />isPay</div>
            <input type="number" v-model="token" @input="checkMax" min="1" placeholder="0" />
          </div>
        </div>
        <!-- <div class="trade-form-item">
          <div class="switch-btn" @click="switchMoney">切换{{ currentMoney === 'usdt' ? 'ISPAY' : 'USDT' }}</div>
        </div> -->
        <!-- <div class="trade-form-item" v-if="currentMoney === 'usdt'">
          <div class="trade-form-label">
            <span>{{ lang('收到') }}</span>
            <span>{{ lang('余额') }}: {{ userinfo.balanceUsdt }}</span>
          </div>
          <div class="trader-form-input">
            <div class="trader-form-input-name"><i class="usdt-icon" />USDT</div>
            <input type="text" placeholder="0" v-model="usdt" readonly />
          </div>
        </div>
        <div class="trade-form-item" v-else>
          <div class="trade-form-label">
            <span>{{ lang('收到') }}</span>
            <span>{{ lang('余额') }}: {{ userinfo.balanceC }}</span>
          </div>
          <div class="trader-form-input">
            <div class="trader-form-input-name"><i class="ispay-icon" />ISPAY</div>
            <input type="text" placeholder="0" v-model="ispay" readonly />
          </div>
        </div> -->
        <!-- <f7-button class="trade-btn" :disabled="!token || token === '0'" preloader :loading="loading" @click="handleExchange()" large fill>{{lang('确认')}}</f7-button> -->
      </div>
    </div>
  </f7-page>
</template>
<script setup lang="ts">
import ChildrenHeader from '../../components/header/childrenHeader.vue'
import userPerson from "@/pinia/person";
import request from "@/tools/request";
import { f7 } from 'framework7-vue'
import { showFailToast, showSuccessToast } from "vant";
import { useI18n } from 'vue-i18n'
import BiwMeta from '@/services/index'
import { $WALLET_AUTHORIZE_ADDRESS_TYPE, $WALLET_PLAOC_PATH, $WALLET_SIGNATURE_TYPE, CHAIN_NAME, type $WEALLET_ADDRESS_RESPONSE } from '@/services/biwmeta/types';
import lang from '@/i18n/index'
const person = userPerson();
const userinfo = $computed(() => person.userinfo);

let loading = $ref(false)
let token: string | null = $ref(null)
let currentMoney: string = $ref('ispay')

const usdt = $computed(() => {
  if (token) {
    return (Number(token) * userinfo.biwPrice - Number(token) * userinfo.biwPrice * userinfo.exchangeRate).toFixed(2)
  }

  return null
});

const ispay = $computed(() => {
  if (token) {
    return (Number(token) * userinfo.priceC - Number(token) * userinfo.priceC * userinfo.exchangeRateC).toFixed(2)
  }

  return null
});

const switchMoney = () => {
  if (currentMoney === 'usdt') {
    currentMoney = 'ispay'
  } else {
    currentMoney = 'usdt'
  }
}

const checkMax = (e: any) => {
  const currentNum = Number(e.target.value)
  const maxNum = Number(userinfo.balanceBiw.replace(/[^0-9\.]/g, ''))

  if (currentNum > maxNum) {
    token = String(maxNum)
  }
}

const setMax = () => {
  if (Number(userinfo.balanceBiw.replace(/[^0-9\.]/g, '')) < 0.1) return
  token = userinfo.balanceBiw.replace(/[^0-9\.]/g, '')
}

// let res: any = await request.get("app_server/user_info");


</script>
<style lang="less" scoped>
.trade {
  width: 90%;
  min-height: 200px;
  margin: 15px auto;
  background: #29313C;
  border-radius: 10px;
  box-shadow: @bs1;
  padding: 15px;
  box-sizing:border-box;
  -moz-box-sizing:border-box;
  -webkit-box-sizing:border-box;
  .trade-title {
    font-weight: 500;
    font-size: 14px;
    margin-bottom: 20px;
    display: flex;
  }
  .trade-table {
    background: #202630;
    margin-bottom: 20px;
    border-radius: 6px;
    ul {
      display: flex;
      padding: 10px 0;
      li {
        flex-grow: 1;
        flex-basis: 0;
        text-align: center;
        p {
          line-height: 1.8;
          &:nth-child(1) {
            color: #CCC;
            font-size: 12px;
          }
        }
      }
    }
  }
  .trade-form {
    display: flex;
    flex-direction: column;
    gap: 20px;
    .trade-form-item {
      display: flex;
      flex-direction: column;
      gap: 10px;
      .switch-btn {
        height: 40px;
        line-height: 40px;
        text-align: center;
        border-radius: 6px;
        background: #202630;
        color: #CCC;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 5px;
        &::before {
          width: 16px;
          height: 16px;
          background: url('data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAQAAADZc7J/AAAAnUlEQVR42u3TMQrCQBAF0IF0QYt0sUojuVFqIeZ/yN7IK1gtNiqIiyArSMyVxtJ+FoLFvumm+M3Ml2xhCIhDK3aIVMx9I1ZDyw8V0dVixQ2fVN7HSqz6NW5UnFmKHU5U+K74LQLVMMfEAIT82P8k/Yw4JAZ0BTwVXuxY4kLl1a3EaqwQqHjsarFyNV5Uvvdbe0SDmcpJ7IaWU27Z4r62a8v8uNFv1gAAAABJRU5ErkJggg==') no-repeat;
          background-size: 16px 16px;
          content: '';
          display: inline-block;
        }
      }
      .trade-form-label {
        display: flex;
        justify-content: space-between;
        font-size: 12px;
        color: #CCC;
      }
      .trader-form-input {
        display: flex;
        height: 42px;
        border-radius: 12px;
        background: #202630;
        input[type="number"]::-webkit-outer-spin-button,
        input[type="number"]::-webkit-inner-spin-button {
            -webkit-appearance: none;
            margin: 0;
        }
        /* 对 Firefox 有效 */
        input[type="number"] {
            -moz-appearance: textfield;
        }
        .trader-form-input-name {
          flex-shrink: 0;
          display: flex;
          align-items: center;
          padding: 0 15px;
          gap: 10px;
          font-weight: 500;
          i {
            width: 32px;
            height: 32px;
            border-radius: 50%;
            display: block;
            overflow: hidden;
            &.biw-icon {
              background: url('../../assets/images/biw-logo.jpg') no-repeat center center;
              background-size: 32px 32px;
            }
            &.usdt-icon {
              background: url('../../assets/images/usdt-icon.png') no-repeat center center;
              background-size: 32px 32px;
            }
            &.ispay-icon {
              background: url('../../assets/images/ispay-icon.png') no-repeat center center;
              background-size: 32px 32px;
            }
          }
        }
        input {
          height: 42px;
          flex-grow: 1;
          text-align: right;
          padding: 0 15px;
        }
      }
    }
  }
  .trade-btn {
    height: 42px;
    border: 0;
    border-radius: 6px;
    background: @lg2;
    font-weight: 500;
    margin-top: 20px;
  }
}
</style>
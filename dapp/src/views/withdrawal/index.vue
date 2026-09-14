<template>
  <div class="withdrawal-page">
    <ChildrenHeader />
    <div class="withdrawal-main">
      <div class="content-box">
        <div class="content-box-title">
          {{lang('提现金额')}}
        </div>
        <!-- 标签栏 -->
        <div class="tab-title">
          <div class="tab-link" :class="{'tab-link-active': tabMenu === 1}" @click="tabMenu = 1">USDT{{ lang('提现') }}</div>
          <div class="tab-link" :class="{'tab-link-active': tabMenu === 2}" @click="tabMenu = 2">ISPAY{{ lang('提现') }}</div>
        </div>

        <!-- 标签内容 -->
        <div class="tabs-container">
          <!-- 标签A内容 -->
          <div class="tab-content" v-if="tabMenu === 1">
            <div class="block">
              <div class="content-box-content">
                <input class="form-input" v-model="amountUsdt" @input="checkUsdtMax" type="number" :placeholder="lang('请输入金额')" />
                <div class="form-sidebar">
                  <button class="all-amount-btn" @click="handleAllAmount()">{{lang('全部')}}</button>
                  <div class="form-balance">{{lang('余额')}}: {{ userinfo.usdt }}</div>
                </div>
              </div>
              <div class="content-box-info">
                <p>{{ lang('最小数量') }}: {{userinfo.withdrawMin}}</p>
                <p>{{ lang('手续费') }}: {{ amountRound(Number(userinfo.withdrawRate) * amountUsdt) }}</p>
              </div>
            </div>
          </div>

          <!-- 标签B内容 -->
          <div class="tab-content" v-if="tabMenu === 2">
            <div class="block">
              <div class="content-box-content">
                <input class="form-input" v-model="amountBiw" @input="checkBiwMax" type="number" :placeholder="lang('请输入金额')" />
                <div class="form-sidebar">
                  <button class="all-amount-btn" @click="handleAllAmount()">{{lang('全部')}}</button>
                  <div class="form-balance">{{lang('余额')}}: {{ userinfo.raw.replace(/[^\d.]/g, '') }}</div>
                </div>
              </div>
              <div class="content-box-info">
                <p>{{ lang('最小数量') }}: {{userinfo.withdrawMinTwo}}</p>
                <p>{{ lang('手续费') }}: {{ amountRound(Number(userinfo.withdrawRateTwo) * amountBiw) }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- 确认按钮 -->
        <button
          v-if="tabMenu === 1"
          class="withdrawal-btn"
          :disabled="!amountUsdt || loading"
          @click="handleWithdrawal()"
        >
          {{ loading ? lang('处理中...') : lang('确认') }}
        </button>
        <button
          v-if="tabMenu === 2"
          class="withdrawal-btn"
          :disabled="!amountBiw || loading"
          @click="handleWithdrawal()"
        >
          {{ loading ? lang('处理中...') : lang('确认') }}
        </button>
      </div>

      <!-- 提现详情列表 -->
      <div class="withdrawal-list">
        <div class="withdrawal-list-title">{{ lang('提现详情') }}</div>
        <div class="withdrawal-list-content">
          <div class="table">
            <div class="table-header">
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
          <div class="empty-data" v-if="amountList.length === 0"></div>
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
</template>
<script setup lang="ts">
import ChildrenHeader from '../../components/header/childrenHeader.vue'
import fetchSign from '../../pinia/fetchSign'
import userPerson from "@/pinia/person";
import request from "@/tools/request";
import { Pagination, showToast } from "vant";
import lang from '@/i18n/index'
import { ref, computed } from 'vue'

const person = userPerson();
const userinfo = computed(() => person.userinfo);
const sign = computed(() => person.sign);

let allPage = ref(1);
let allPageCount = ref(1);
let amountUsdt: any = ref('')
let usdtSite: string = ref('')
let ispaySite: string = ref('')
let amountBiw: any = ref('')
let amountISPay: any = ref('')
let amount: any = ref('')
let tabMenu: number = ref(1)
let loading = ref(false)
let amountList: any[] = ref([])

const amountRound = (num: number) => {
  return Math.round(num * 100) / 100
}

const getAmountList = async (page: number = 1) => {
    await request.get("app_server/withdraw_list", {
      params: {
        page
      }
    }).then((res: any) => {
      allPageCount.value = Math.ceil(res.count / 10);
      amountList.value = res.list
    })
}

getAmountList()

const handleAllAmount = () => {
  if (tabMenu.value === 1) {
    if (Number(userinfo.value.usdt) < 0.1) return false
    amountUsdt.value = userinfo.value.usdt
  }
  if (tabMenu.value === 2) {
    const biw = userinfo.value.raw.replace(/[^\d.]/g, '')
    if (Number(biw) < 0.1) return false
    amountBiw.value = biw
  }
}

const checkUsdtMax = (e: any) => {
  const currentNum = Number(e.target.value)
  const maxNum = Number(userinfo.value.usdt)

  if (currentNum > maxNum) {
    amountUsdt.value = maxNum
  }
}

const checkBiwMax = (e: any) => {
  const currentNum = Number(e.target.value)
  const maxNum = Number(userinfo.value.raw)

  if (currentNum > maxNum) {
    amountBiw.value = maxNum
  }
}

const handleWithdrawal = async() => {
  if (tabMenu.value !== 1) {
    showToast({
      message: lang("暂未开通"),
      position: 'center',
      duration: 2000,
    });
    return
  }
  if (loading.value) return
  loading.value = true

  await request.post("app_server/withdraw", {
    amount: tabMenu.value === 1 ? amountUsdt.value : amountBiw.value,
    sign: sign.value
  }).then((res: any) => {
    loading.value = false

    if (res.status === 'ok') {
      showToast({
        message: lang("提现成功"),
        position: 'center',
        duration: 2000,
      });
    } else {
      showToast({
        message: res.status,
        position: 'center',
        duration: 2000,
      });
		}
  }).catch((err) => {
    loading.value = false
    showToast({
      message: '提现失败',
      position: 'center',
      duration: 2000,
    });
  })
}

</script>
<style scoped lang="less">
@import "./styles/index.less";

.withdrawal-page {
  width: 100%;
  min-height: 100vh;
}

/* 标签栏样式 */
.tab-title {
  display: flex;
  justify-content: space-around;
  align-items: center;
  background-color: #29313C;
  padding: 10px 0;
  margin: 10px 0;
}

.tab-link {
  color: #CCC;
  padding: 10px 20px;
  cursor: pointer;
  transition: all 0.3s;
}

.tab-link-active {
  color: #FCD434;
  font-weight: bold;
}

/* 标签内容容器 */
.tabs-container {
  margin-bottom: 20px;
}

.tab-content {
  padding: 10px;
}

/* 块样式 */
.block {
  padding: 10px;
  background-color: #333;
  border-radius: 8px;
  margin-bottom: 10px;
}

/* 按钮样式 */
.all-amount-btn {
  padding: 5px 15px;
  background-color: transparent;
  border: 1px solid #FCD434;
  color: #FCD434;
  border-radius: 15px;
  cursor: pointer;
  font-size: 14px;
  margin-right: 10px;
}

.withdrawal-btn {
  width: 100%;
  padding: 12px;
  background-color: #FCD434;
  color: #000;
  border: none;
  border-radius: 4px;
  font-size: 16px;
  font-weight: bold;
  cursor: pointer;
  transition: all 0.3s;
}

.withdrawal-btn:disabled {
  background-color: #999;
  cursor: not-allowed;
}
</style>

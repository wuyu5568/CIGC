<template>
  <div class="index-page">
    <div class="header" v-if="menu === 1">
      <div class="header-main">
        <div class="header-logo">ISPAY</div>
        <div class="header-right">
          <div class="user-address">{{ formatAddress(address) }}</div>
          <!-- <div class="lang-btn"></div> -->
          <a-dropdown :trigger="['click']">
            <div class="lang-btn"></div>
            <template #overlay>
              <a-menu @click="onLangSelect">
                <a-menu-item key="zh" :class="{'selected-lan': getLan() === 'zh'}">简体中文</a-menu-item>
                <a-menu-item key="en" :class="{'selected-lan': getLan() === 'en'}">English</a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
          <i class="menu-button" @click="show = true"></i>
        </div>
      </div>
    </div>
    <!-- <ChildrenHeader :handleMenuChange="handleMenuChange" v-else /> -->
    <div class="swiper-container">
      <van-swipe :autoplay="3000" indicator-color="#fff">
        <van-swipe-item><img :src="image1" style="width: 100%" /></van-swipe-item>
        <van-swipe-item><img :src="image2" style="width: 100%" /></van-swipe-item>
        <van-swipe-item><img :src="image3" style="width: 100%" /></van-swipe-item>
      </van-swipe>
    </div>
    <div class="main-body">
      <div class="board-box">
        {{ lang('活动即将开始，欢迎关注IsPay官方平台。推广并分享个人推广码') }}
        <!-- <a-carousel autoplay dot-position="left" :dots="false" :autoplaySpeed="3000">
          <div v-for="(item, index) in notices" :key="index">
            <p><NotificationOutlined style="margin-right: 10px" />{{ item }}</p>
          </div>
        </a-carousel> -->
      </div>
      <div class="show-image">
        <img :src="image4" />
      </div>
      <div class="show-text">
        <h2>{{ lang('结合人工智能与区块链先进科技，驱动行业创新') }}</h2>

        <h3>1. {{ lang('代币经济模型') }}：</h3>
        <p>{{ lang('发行总量 10 亿枚') }}</p>

        <h3>2. {{ lang('输出机制') }}：</h3>
        <p>{{ lang('块输出间隔：') }}3 seconds/block</p>
        <p>{{ lang('每天块数：') }}28800</p>

        <h3>3. {{ lang('实用场景') }}：</h3>
        <ul>
          <li>{{ lang('支付网络交易费用') }}</li>
          <li>{{ lang('参与生态系统治理投票') }}</li>
          <li>{{ lang('访问链上服务') }}</li>
          <li>{{ lang('作为交易对基础资产科技架构') }}</li>
        </ul>

        <h3>4. {{ lang('技术架构') }}：</h3>

        <p>
          <strong>{{ lang('人工智能：') }}</strong>
          {{ lang('WAI 系统运用深度及强化学习技术，能自我优化并分析区块链数据，自动调整以提供更智能的服务。') }}
        </p>

        <p>
          <strong>{{ lang('量子共识：') }}</strong>
          {{ lang('我们创新的共识机制融合了量子计算的安全性与人工智能的协作能力，强化了网络防御力，孕育出一个既强大又高度适应的生态系统。') }}
        </p>

        <p>
          <strong>{{ lang('智能平台：') }}</strong>
          {{ lang('WSP 平台用 AI 智能融合的开发，通过语义分析增强性能与安全，支持多语言程序设计，满足开发者需求。') }}
        </p>

        <p>
          <strong>{{ lang('分析式存储：') }}</strong>
          {{ lang('我们的平台汇聚了众多开发者与企业，促进了人工智能领域专业人士、企业家及合作伙伴间的紧密合作，共同推动科技不断前行。') }}
        </p>
      </div>
      <div class="show-image">
        <img :src="image3" />
      </div>
      <div class="show-image">
        <img :src="image5" />
      </div>
      <div class="show-text">
        <ol>
          <li>
            <strong>{{ lang('ISPAY 卡支付') }}：</strong><br />
            {{ lang('提供连通数字货币与全球消费的合规支付，实现“消费即挖矿”。') }}
          </li>

          <li>
            <strong>{{ lang('ISPAY 公链') }}：</strong><br />
            {{ lang('构建了支持多链结构的底层基础设施，确保生态安全与互操作性。') }}
          </li>

          <li>
            <strong>{{ lang('ISPAY 社交') }}：</strong><br />
            {{ lang('打造一个保护隐私且能通过互动获得激励的“边聊边赚”平台。') }}
          </li>

          <li>
            <strong>{{ lang('ISPAY 钱包') }}：</strong><br />
            {{ lang('作为生态入口，提供安全便捷的多链资产管理和 DApp 使用体验。') }}
          </li>

          <li>
            <strong>{{ lang('ISPAY 交易所') }}：</strong><br />
            {{ lang('提供高效合规的去中心化加密资产交易，并创新“带股权”模式。') }}
          </li>

          <li>
            <strong>{{ lang('ISPAY SWAP') }}：</strong><br />
            {{ lang('是一个支持 DAO 治理的去中心化交易所，实现安全自主的链上交易。') }}
          </li>

          <li>
            <strong>{{ lang('ISPAY 商城') }}：</strong><br />
            {{ lang('支持加密货币支付并融合“消费挖矿”机制的 Web3 电商平台。') }}
          </li>

          <li>
            <strong>{{ lang('ISPAY 链游') }}：</strong><br />
            {{ lang('通过《矿晶农场》等游戏构建“游戏即收益”的互动元宇宙生态。') }}
          </li>
        </ol>

        <p>
          <strong>{{ lang('总结') }}：</strong>
          {{ lang('ISPAY 生态通过八大核心模块，构建了一个从基础设施到应用场景的一站式 Web3 服务平台。') }}
        </p>
      </div>
      <div class="footer">
        <a href="/ISPAY%20Chain%20White%20Paper-1.pdf">{{ lang('下載英文白皮書') }}</a>
        <a href="/ISPAY%20Chain%20White%20Paper-2.pdf">{{ lang('下載中文白皮書') }}</a>
      </div>
    </div>
    <div class="link">
      <div>{{ lang('生态') }}</div>
      <div>
        <a href="https://ispay.com/home/index?utm_source=tokenpocket"><img src="https://mall.ispay.club/uploads/20251122/f16bfe3ac0bff28ebcf7419ed658d584.png" draggable="false"></a>
        <a href="https://scan.ispay.vip/zh-CN?utm_source=tokenpocket"><img src="https://mall.ispay.club/uploads/20251121/7d67e20a91269121619a2f18af27b6d3.png" draggable="false"></a>
        <a href="https://web.ispay.vip/web/?utm_source=tokenpocket#/"><img src="https://mall.ispay.club/uploads/20251121/f403271cbb03c8aa7406fd547107a42e.png" draggable="false"></a>
        <a href="https://wallet.ispay.vip/?utm_source=tokenpocket"><img src="https://mall.ispay.club/uploads/20251121/752dea1343a0ecd3720ed2877b5d5bcb.png" draggable="false"></a>
        <a href="https://www.playgamefrane.com/game/"><img src="https://mall.ispay.club/uploads/20251121/b6b194f3d91d1d6139a9879689931c17.png" draggable="false"></a>
      </div>
    </div>
    <van-popup v-model:show="show" overlay-class="sidebar-popup" position="right">
      <Sidebar />
    </van-popup>
  </div>
</template>
<script setup lang="ts">
import { NotificationOutlined } from '@ant-design/icons-vue'
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import copy from 'copy-to-clipboard';
import ChildrenHeader from '../../components/header/childrenHeader.vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
const { locale } = useI18n()
import Sidebar from './components/sidebar.vue';
import image1 from '@/assets/images/a83da2dd0025525a207be0521e958705.jpg'
import image2 from '@/assets/images/7a742d3bdfd90221022944759b5bbd52.jpg'
import image3 from '@/assets/images/569e5ba10826da3e0713cbefc9a59287.jpg'
import image4 from '@/assets/images/8ef11d0f7020b02955f78daf4abe3702.jpg'
import image5 from '@/assets/images/c76a2c706c5bb610078df71c72d46344.jpg'

const router = useRouter()
let show = ref(false)
let menu = ref(1)
let isOpened = ref(false)
const person = userPerson();
const address = computed(() => person.address);
const userinfo = computed(() => person.userinfo);

const handleMenuChange = (index: number) => {
  menu = index
}

const notices = ref([userinfo.notice]);

const copyToClipboard = (text: string) => {
  copy(`${window.location.origin}?code=${text}`);
  showToast({
    message: lang("内容已复制到剪贴板"),
    position: 'center',
    duration: 2000,
  });
}

const openPayment = () => {
  router.push('payment')
}

const openWindow = (url: string) => {
  window.open(url)
}

const formatAddress = (value: string) => {
  const frontSix = value.slice(0, 6);
  const backSix = value.slice(-6);
  const middle = '...';
  return frontSix + middle + backSix;
}

const onLangSelect = (event: any) => {
  const key = event?.key;
  if (key === 'en') {
    localStorage.setItem('lan', 'en')
    locale.value = 'en'
  } else if (key === 'zh') {
    localStorage.setItem('lan', 'zh')
    locale.value = 'zh'
  }
}
const getLan = () => window.localStorage.getItem('lan') || 'zh'

</script>
<style lang="less">
.ant-dropdown-menu-item {
  font-size: 14px!important;
}
.selected-lan {
  color: #3640f0!important;
}
</style>
<style scoped lang="less">
@import "index";

.index-page {
  width: 100%;
  min-height: 100vh;
  position: relative;
}

.swiper-container {
  width: 100%;
  height: 200px;
}
</style>

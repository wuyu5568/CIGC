<template>
  <div class="home">
    <img :src="indexEnBg" v-if="language === 'en'" />
    <img :src="indexBg" v-else />
  </div>
</template>
<script setup lang="ts">
import banner from '../../../assets/images/banner.png'
import Exchange from './exchange.vue'
import userPerson from "@/pinia/person";
import dayjs from 'dayjs'
import lang from '@/i18n/index'
import RollingNew from './rollingNew.vue';
import { defineProps, defineEmits } from 'vue'
import indexBg from '../../../assets/images/20250522/index_bg.png'
import indexEnBg from '../../../assets/images/20250522/index-en_bg.png'
import { useI18n } from 'vue-i18n'
import { ref, computed } from 'vue'
const { locale } = useI18n()

const language = locale.value
const person = userPerson();
const userinfo = computed(() => person.userinfo);
let showExchange = ref(false)

const props = defineProps({
  handleMenuChange: Function
})

const countDown = computed(() => {
  return getCountDown(userinfo.time)
});

const formatAddress = (value: string) => {
  const frontSix = value.slice(0, 6);
  const backSix = value.slice(-4);
  const middle = '...';
  return frontSix + middle + backSix;
}

const getCountDown = (time: number) => {
  const now = dayjs(time * 1000);
  // 获取今天结束的时间点
  const todayEnd = now.endOf('day').add(1, 'second');
  // 计算剩余的时间
  const duration = todayEnd.diff(now); // 返回毫秒数
  // 将剩余时间转换成小时和分钟
  const hours = parseInt(`${duration / 3600000}`); // 毫秒数转换成小时
  const minutes = parseInt(`${(duration - Math.floor(hours) * 3600000) / 60000}`); // 剩余毫秒数转换成分钟

  return `${hours < 10 ? '0' + hours : hours}:${minutes < 10 ? '0' + minutes : minutes}`
}

</script>
<style scoped lang="less">
@import "../styles/home.less";
</style>

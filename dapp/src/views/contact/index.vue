<template>
  <f7-page name="node">
    <ChildrenHeader />
    <div class="contact">
      <img v-if="language === 'en'" :src="contactEn" />
      <img v-else :src="contact" />
    </div>
  </f7-page>
</template>
<script setup lang="ts">
import ChildrenHeader from '../../components/header/childrenHeader.vue'
import contact from '@/assets/images/20250522/contact.png'
import contactEn from '@/assets/images/20250522/whitePaper-en_bg.png'
import { f7 } from 'framework7-vue'
import lang from '@/i18n/index'
import userPerson from "@/pinia/person";
import { useI18n } from 'vue-i18n'
const { locale } = useI18n()

const language = locale.value
const person = userPerson();
const userinfo = $computed(() => person.userinfo);

let toast: any = $ref(null)

</script>
<style scoped lang="less">
@import "./styles/index.less";
</style>

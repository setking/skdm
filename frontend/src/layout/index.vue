<script setup lang="ts">
import Footer from "@/components/footer/index.vue"
import Header from "@/components/header/index.vue"
import Menu from "@/components/menu/index.vue"
import AppMain from "./main/index.vue"

</script>

<template>
  <div class="layout">
    <div class="header">
      <Header/>
    </div>
    <div class="content">
      <div class="container">
        <div class="side-bar">
          <n-message-provider>
            <Menu/>
          </n-message-provider>
        </div>
        <div class="main">
          <AppMain/>
        </div>
      </div>
    </div>
<!--    <div class="footer">-->
<!--      <Footer/>-->
<!--    </div>-->
  </div>
</template>
<style scoped lang="scss">
.layout {
  width: 100vw;
  height: 100vh;
  display: grid;
  grid-template-rows: auto 1fr;   /* header / content */
  overflow: hidden;
}
.content {
  overflow: hidden;
  .container {
    height: 100%;
    display: flex;
    justify-content: flex-start;
    .side-bar{
      border-right: 1px solid #ddd;
    }
    .main {
      flex: 1;
      min-width: 0;
      overflow: hidden;
    }
  }
}
</style>

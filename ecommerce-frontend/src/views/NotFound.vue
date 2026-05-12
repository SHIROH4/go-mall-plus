<template>
  <div class="not-found-page">
    <div class="content">
      <div class="error-code">404</div>
      <h1>页面不存在</h1>
      <p>抱歉，您访问的页面不存在或已被删除</p>
      <div class="actions">
        <el-button type="primary" size="large" @click="$router.push('/home')">
          返回首页
        </el-button>
        <el-button size="large" @click="$router.back()">
          返回上页
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.not-found-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.content {
  text-align: center;
}

.error-code {
  font-size: 120px;
  font-weight: bold;
  opacity: 0.3;
  line-height: 1;
  margin-bottom: 20px;
}

.content h1 {
  font-size: 32px;
  margin-bottom: 12px;
}

.content p {
  font-size: 16px;
  opacity: 0.8;
  margin-bottom: 32px;
}

.actions {
  display: flex;
  gap: 16px;
  justify-content: center;
}
</style>

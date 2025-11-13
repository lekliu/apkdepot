<script setup>
import { RouterView, RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
const authStore = useAuthStore()
</script>

<template>
  <header class="navbar">
    <div class="container nav-container">
      <router-link to="/" class="brand">ApkDepot</router-link>
      <nav>
        <router-link to="/">Home</router-link>
        <router-link v-if="authStore.isLoggedIn" to="/upload">Upload</router-link>
        <router-link v-if="!authStore.isLoggedIn" to="/login">Login</router-link>
        <a href="#" v-if="authStore.isLoggedIn" @click.prevent="authStore.logout()">Logout</a>
      </nav>
    </div>
  </header>
  <main>
    <RouterView />
  </main>
</template>

<style>
@import '@/assets/main.css';

.navbar {
  background: #fff;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  padding: 0.5rem 0;
  margin-bottom: 2rem;
}
.nav-container {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.brand {
  font-size: 1.5rem;
  font-weight: bold;
  text-decoration: none;
  color: #0056b3;
}
nav a {
  margin-left: 1rem;
  text-decoration: none;
  color: #333;
}
nav a.router-link-exact-active {
  font-weight: bold;
  color: #007bff;
}
</style>

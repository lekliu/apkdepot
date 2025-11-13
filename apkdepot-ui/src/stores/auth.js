import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import axios from 'axios';
import router from '@/router';

/**
 * Pinia Store for Authentication Management.
 *
 * This store handles:
 * - Storing the JWT token.
 * - Persisting the token to localStorage.
 * - Providing login status via a computed getter.
 * - Handling login and logout actions.
 * - Automatically setting the Authorization header for all Axios requests.
 */
export const useAuthStore = defineStore('auth', () => {
  // --- STATE ---
  // The token is initialized from localStorage, allowing the user's session
  // to persist across page refreshes.
  const token = ref(localStorage.getItem('token') || null);

  // --- GETTERS ---
  // A computed property that returns true if a token exists, false otherwise.
  // This is used throughout the app to determine if the user is logged in.
  const isLoggedIn = computed(() => !!token.value);

  // --- ACTIONS ---

  /**
   * Sets the authentication token, updates localStorage, and configures
   * the default Authorization header for all subsequent Axios requests.
   * @param {string | null} newToken - The new JWT token, or null to clear it.
   */
  function setToken(newToken) {
    token.value = newToken;
    if (newToken) {
      // If a token is provided, store it in localStorage and set the Axios header.
      localStorage.setItem('token', newToken);
      axios.defaults.headers.common['Authorization'] = `Bearer ${newToken}`;
    } else {
      // If the token is null, remove it from localStorage and delete the Axios header.
      localStorage.removeItem('token');
      delete axios.defaults.headers.common['Authorization'];
    }
  }

  /**
   * Performs a login request to the backend API.
   * On success, it sets the new token.
   * Throws an error on failure, which can be caught in the component.
   * @param {object} credentials - An object containing { username, password }.
   */
  async function login(credentials) {
    // The component calling this action will handle the try-catch block.
    const response = await axios.post('/api/login', credentials);
    const newToken = response.data.token;
    if (!newToken) {
      throw new Error("No token received from server");
    }
    setToken(newToken);
  }

  /**
   * Logs the user out by clearing the token and redirecting to the login page.
   */
  function logout() {
    setToken(null);
    router.push('/login');
  }

  // --- INITIALIZATION ---
  // When the store is initialized (e.g., on page load), if a token already
  // exists in localStorage, ensure the Axios header is set correctly.
  if (token.value) {
    setToken(token.value);
  }

  // Expose state, getters, and actions to be used in components.
  return {
    token,
    isLoggedIn,
    login,
    logout,
  };
});

<script setup>
import { ref, onMounted, computed, onUnmounted } from 'vue'

const API = '/api/proxy/v1'
const authenticated = ref(false)
const authChecking = ref(true)
const authPassword = ref('')
const authError = ref('')

const images = ref([])
const tags = ref([])
const featuredTags = ref([])
const categories = ref([])
const selectedTag = ref('')
const selectedCategory = ref('')
const orientation = ref('')
const mode = ref('random') // 'random' | 'tag' | 'gallery' | 'gallery-detail'
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(true)
const error = ref('')
const modalImg = ref(null)
const modalMeta = ref(null)
const showModal = ref(false)
const searchQuery = ref('')
const showDropdown = ref(false)
const zoom = ref(1)
const panX = ref(0)
const panY = ref(0)
const isPanning = ref(false)
const startX = ref(0)
const startY = ref(0)
const lastDist = ref(0)

// Gallery state
const galleries = ref([])
const selectedGallery = ref(null)
const galleryPage = ref(1)
const galleryTotalPages = ref(1)

const filteredTags = computed(() => {
  const list = tags.value
  if (!list.length) return []
  if (!searchQuery.value) return list
  const q = searchQuery.value.toLowerCase()
  return list.filter(t => (t.name || t).toLowerCase().includes(q))
})

const tagCount = computed(() => tags.value.length)

// Active filters for chips display
const activeFilters = computed(() => {
  const chips = []
  if (selectedTag.value) chips.push({ type: 'tag', label: selectedTag.value, icon: '#' })
  if (selectedCategory.value) chips.push({ type: 'category', label: selectedCategory.value, icon: '📂' })
  if (orientation.value) {
    const labels = { portrait: '竖图', landscape: '横图' }
    chips.push({ type: 'orientation', label: labels[orientation.value] || orientation.value, icon: '↕' })
  }
  return chips
})

let observer = null
const sentinel = ref(null)

onMounted(async () => {
  // Already authenticated this session
  if (sessionStorage.getItem('gallery_auth') === '1') {
    authenticated.value = true
    authChecking.value = false
    await loadGallery()
    return
  }
  // Try no-password auto-auth (backend accepts any request when ACCESS_PASSWORD is empty)
  try {
    const res = await fetch('/api/auth', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: '' })
    })
    if (res.ok) {
      sessionStorage.setItem('gallery_auth', '1')
      authenticated.value = true
      authChecking.value = false
      await loadGallery()
      return
    }
  } catch {}
  // Password required — show auth modal
  authChecking.value = false
})

async function loadGallery() {
  await Promise.all([fetchTags(), fetchFeaturedTags(), fetchCategories()])
  if (featuredTags.value.length > 0) {
    selectedTag.value = featuredTags.value[0].name || featuredTags.value[0]
    mode.value = 'tag'
    loadImages()
  } else {
    mode.value = 'random'
    loadRandom()
  }
}

async function doAuth() {
  authError.value = ''
  try {
    const res = await fetch('/api/auth', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: authPassword.value })
    })
    if (res.ok) {
      sessionStorage.setItem('gallery_auth', '1')
      authenticated.value = true
      await loadGallery()
    } else {
      authError.value = '密码错误'
    }
  } catch {
    authError.value = '验证失败，请重试'
  }
}

onUnmounted(() => { if (observer) observer.disconnect() })

function observeScroll() {
  setTimeout(() => {
    if (observer) observer.disconnect()
    const el = document.getElementById('scroll-sentinel')
    if (!el) return
    observer = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting && hasMore.value && !loadingMore.value && !loading.value) {
        loadMore()
      }
    }, { rootMargin: '300px' })
    observer.observe(el)
  }, 100)
}

async function fetchTags() {
  try { const d = await (await fetch(`${API}/tags`)).json(); tags.value = d.items || d.tags || d || [] }
  catch { tags.value = [] }
}

async function fetchFeaturedTags() {
  try { const d = await (await fetch(`${API}/featured-tags`)).json(); featuredTags.value = d.items || d.tags || d || [] }
  catch { featuredTags.value = [] }
}

async function fetchCategories() {
  try { const d = await (await fetch(`${API}/categories`)).json(); categories.value = d.items || d.categories || d || [] }
  catch { categories.value = [] }
}

async function loadImages() {
  loading.value = true; error.value = ''; images.value = []; hasMore.value = false
  if (!selectedTag.value) { hasMore.value = true; mode.value = 'random'; loadRandom(); return }
  try {
    const d = await (await fetch(`${API}/tag/${encodeURIComponent(selectedTag.value)}/preview`)).json()
    const ids = d.image_ids || d.ids || []
    images.value = ids.map(id => ({ id, url: `${API}/image/${id}`, metaUrl: `${API}/image/${id}/meta` }))
    hasMore.value = true
  } catch { error.value = 'Failed to load images' }
  loading.value = false
  observeScroll()
}

async function loadRandom() {
  loading.value = true; error.value = ''; images.value = []; hasMore.value = true
  loading.value = false
  await appendRandom()
}

async function appendRandom() {
  if (loadingMore.value) return
  loadingMore.value = true
  const p = new URLSearchParams({ _t: Date.now() })
  if (selectedTag.value) p.set('tag', selectedTag.value)
  if (selectedCategory.value) p.set('category', selectedCategory.value)
  if (orientation.value) p.set('orientation', orientation.value)
  try {
    const resp = await fetch(`${API}/random?${p}`)
    if (!resp.ok) { loadingMore.value = false; return }
    const id = 'rand-' + Date.now()
    images.value.push({ id, url: `${API}/random?${p}`, metaUrl: null })
    hasMore.value = true
  } catch { hasMore.value = false }
  loadingMore.value = false
  observeScroll()
}

async function loadMore() {
  if (mode.value === 'gallery' && selectedGallery.value) {
    galleryPage.value++
    loadGalleryPage(galleryPage.value)
    return
  }
  await appendRandom()
}

function selectTag(tag) {
  selectedTag.value = tag.name || tag; selectedCategory.value = ''
  mode.value = 'tag'
  searchQuery.value = tag.name || tag; showDropdown.value = false
  loadImages()
}

function selectCategory(cat) {
  selectedCategory.value = cat.name || cat; selectedTag.value = ''; searchQuery.value = ''
  mode.value = 'random'
  loadRandom()
}

function toggleOrientation(o) {
  orientation.value = orientation.value === o ? '' : o
  if (mode.value === 'tag' && selectedTag.value) { images.value = []; loadImages() }
  else { images.value = []; loadRandom() }
}

function clearFilter(type) {
  if (type === 'tag') { selectedTag.value = ''; mode.value = 'random'; loadRandom() }
  else if (type === 'category') { selectedCategory.value = ''; (mode.value === 'tag' && selectedTag.value) ? loadImages() : loadRandom() }
  else if (type === 'orientation') { orientation.value = ''; (mode.value === 'tag' && selectedTag.value) ? loadImages() : loadRandom() }
}

function selectMode(m) {
  if (m === 'random') {
    mode.value = 'random'; selectedTag.value = ''; selectedCategory.value = ''; searchQuery.value = ''
    selectedGallery.value = null; images.value = []
    loadRandom()
  } else if (m === 'gallery') {
    mode.value = 'gallery'; selectedTag.value = ''; selectedCategory.value = ''; searchQuery.value = ''
    images.value = []; loading.value = true; error.value = ''
    galleryPage.value = 1; galleryTotalPages.value = 1
    loadRandomGallery()
  }
}

function loadRandomGallery() {
  mode.value = 'gallery'
  loading.value = true; error.value = ''; images.value = []
  fetch(`${API}/gallery/random`).then(r => r.json()).then(d => {
    selectedGallery.value = { id: d.id, title: d.title, name: d.name }
    galleryPage.value = 1
    const pagination = d.images_pagination || {}
    galleryTotalPages.value = pagination.total_pages || 1
    const raw = d.images || d.items || []
    images.value = raw.map(item => {
      const imgId = item.id || item.image_id
      return { id: imgId, url: `${API}/image/${imgId}`, metaUrl: `${API}/image/${imgId}/meta` }
    })
    loading.value = false
  }).catch(() => { error.value = 'Failed to load gallery'; loading.value = false })
}

function loadGalleryPage(page) {
  if (!selectedGallery.value) return
  const id = selectedGallery.value.id
  const pageParam = page > 1 ? `?page=${page}` : ''
  loading.value = true
  fetch(`${API}/gallery/${id}${pageParam}`).then(r => r.json()).then(d => {
    const pagination = d.images_pagination || {}
    galleryTotalPages.value = pagination.total_pages || 1
    const raw = d.images || d.items || []
    const mapped = raw.map(item => {
      const imgId = item.id || item.image_id
      return { id: imgId, url: `${API}/image/${imgId}`, metaUrl: `${API}/image/${imgId}/meta` }
    })
    if (page === 1) {
      images.value = mapped
    } else {
      images.value.push(...mapped)
    }
    hasMore.value = galleryPage.value < galleryTotalPages.value
    loading.value = false
  }).catch(() => { error.value = 'Failed to load gallery'; loading.value = false })
}

function refreshGallery() {
  loadRandomGallery()
}

function backToGalleries() {
  loadRandomGallery()
}

function resetZoom() { zoom.value = 1; panX.value = 0; panY.value = 0 }

function openModal(img) {
  modalImg.value = img; modalMeta.value = null; showModal.value = true; resetZoom()
  if (img.metaUrl) fetch(img.metaUrl).then(r => r.json()).then(d => modalMeta.value = d).catch(() => {})
}

function closeModal() { showModal.value = false; modalImg.value = null; modalMeta.value = null; resetZoom() }

function handleWheel(e) {
  if (e.ctrlKey || e.metaKey) { e.preventDefault(); zoom.value = Math.max(1, Math.min(5, zoom.value + (e.deltaY > 0 ? -0.1 : 0.1))) }
}
function touchStartImg(e) {
  if (e.touches.length === 2) lastDist.value = Math.hypot(e.touches[0].clientX - e.touches[1].clientX, e.touches[0].clientY - e.touches[1].clientY)
}
function touchMoveImg(e) {
  if (e.touches.length === 2) { e.preventDefault();
    const d = Math.hypot(e.touches[0].clientX - e.touches[1].clientX, e.touches[0].clientY - e.touches[1].clientY)
    zoom.value = Math.max(1, Math.min(5, zoom.value * (d / lastDist.value))); lastDist.value = d
  }
}
function handleDblClick() {
  zoom.value = zoom.value > 1 ? 1 : 2.5
  if (zoom.value === 1) { panX.value = 0; panY.value = 0 }
}
function panStart(e) {
  if (zoom.value <= 1) return; isPanning.value = true
  startX.value = e.clientX - panX.value; startY.value = e.clientY - panY.value
}
function panMove(e) {
  if (!isPanning.value) return
  panX.value = e.clientX - startX.value; panY.value = e.clientY - startY.value
}
function panEnd() { isPanning.value = false }
</script>

<template>
  <div class="min-h-screen bg-gray-950 text-gray-100 font-sans antialiased">
    <!-- Auth Modal -->
    <div v-if="!authenticated && !authChecking" class="fixed inset-0 z-50 bg-gray-950 flex items-center justify-center p-4">
      <div class="w-full max-w-sm bg-gray-900 border border-gray-800 rounded-2xl p-8 shadow-2xl shadow-black/40">
        <div class="text-center mb-6">
          <h1 class="text-2xl font-bold font-display text-white tracking-tight">Gallery</h1>
          <p class="text-gray-400 text-sm mt-1">Enter access password</p>
        </div>
        <form @submit.prevent="doAuth" class="space-y-4">
          <div>
            <label for="auth-password" class="sr-only">Password</label>
            <input id="auth-password" v-model="authPassword" type="password" placeholder="Password" class="w-full px-4 py-2.5 bg-gray-800 border border-gray-700 rounded-xl text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500/40 transition-colors duration-200" autofocus />
          </div>
          <p v-if="authError" class="text-red-400 text-xs text-center">{{ authError }}</p>
          <button type="submit" class="w-full py-2.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl text-sm font-medium transition-colors duration-200 cursor-pointer">Enter</button>
        </form>
      </div>
    </div>

    <!-- App -->
    <template v-if="authenticated">
    <header class="sticky top-0 z-30 mx-2 sm:mx-4 mt-2 rounded-2xl bg-gray-950/80 backdrop-blur-xl border border-gray-800/60 shadow-lg shadow-black/20">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 py-3 sm:py-4">
        <!-- Row 1: Logo + Search + Orientation -->
        <div class="flex flex-col sm:flex-row sm:items-center gap-3 sm:gap-4">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <h1 class="text-xl sm:text-2xl font-bold font-display text-white shrink-0 tracking-tight">Gallery</h1>
            <div class="relative flex-1 min-w-0 sm:min-w-[200px] max-w-sm">
              <label for="tag-search" class="sr-only">Search tags</label>
              <input id="tag-search" v-model="searchQuery" @focus="showDropdown = true" @blur="setTimeout(() => showDropdown = false, 200)" @keydown.enter="filteredTags.length && selectTag(filteredTags[0])" placeholder="Search tag..." class="w-full pl-9 pr-3 py-2 bg-gray-800/80 border border-gray-700/60 rounded-xl text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500/40 transition-colors duration-200" />
              <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-4.35-4.35M11 19a8 8 0 100-16 8 8 0 000 16z" /></svg>
              <div v-if="showDropdown && filteredTags.length" class="absolute top-full mt-1.5 left-0 right-0 bg-gray-800 border border-gray-700/60 rounded-xl max-h-72 overflow-y-auto z-50 shadow-2xl shadow-black/40">
                <div class="sticky top-0 bg-gray-800/95 backdrop-blur px-4 py-1.5 text-xs text-gray-500 border-b border-gray-700/40 flex items-center gap-2">
                  <span v-if="searchQuery">{{ filteredTags.length }} match</span>
                  <span v-else>Available tags · {{ filteredTags.length }}</span>
                </div>
                <button v-for="tag in filteredTags" :key="tag.name || tag" @mousedown.prevent="selectTag(tag)" class="w-full text-left px-4 py-2.5 text-sm hover:bg-gray-700/60 text-gray-300 hover:text-white transition-colors duration-150 flex items-center gap-2">
                  <svg class="w-3.5 h-3.5 text-gray-500 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z" /></svg>
                  <span class="truncate flex-1">{{ tag.name || tag }}</span>
                  <span v-if="tag.gallery_count" class="text-gray-500 text-xs tabular-nums shrink-0">{{ tag.gallery_count }}</span>
                </button>
              </div>
            </div>
          </div>
          <!-- Orientation filter - bigger, more prominent -->
          <div class="flex items-center gap-2 self-start sm:self-auto">
            <span class="text-xs text-gray-500 font-medium uppercase tracking-wider shrink-0">方向</span>
            <div class="flex gap-1 bg-gray-800/60 rounded-xl p-1" aria-label="Orientation filter">
              <button @click="toggleOrientation('portrait')" :class="orientation==='portrait' ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/20 ring-1 ring-indigo-500/30' : 'text-gray-400 hover:text-gray-200 hover:bg-gray-700/50'" class="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-150 min-h-[38px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/60" aria-label="Portrait">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m0 0l-4-4m4 4l4-4" /></svg>
                <span>竖</span>
              </button>
              <button @click="toggleOrientation('landscape')" :class="orientation==='landscape' ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/20 ring-1 ring-indigo-500/30' : 'text-gray-400 hover:text-gray-200 hover:bg-gray-700/50'" class="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-150 min-h-[38px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/60" aria-label="Landscape">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 12H4m0 0l4-4m-4 4l4 4" /></svg>
                <span>横</span>
              </button>
              <button @click="toggleOrientation('')" :class="!orientation ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/20 ring-1 ring-indigo-500/30' : 'text-gray-400 hover:text-gray-200 hover:bg-gray-700/50'" class="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-150 min-h-[38px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/60" aria-label="All orientations">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" /></svg>
                <span>全部</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Active Filter Chips -->
        <div v-if="activeFilters.length" class="flex flex-wrap items-center gap-1.5 mt-3">
          <span class="text-xs text-gray-500 font-medium mr-0.5">筛选：</span>
          <span v-for="chip in activeFilters" :key="chip.type" class="inline-flex items-center gap-1 px-2.5 py-1 bg-gray-800 border border-gray-700/60 rounded-lg text-xs text-gray-200 font-medium">
            <span>{{ chip.icon === '#' ? '#' : chip.icon === '↕' ? (chip.label === '竖图' ? '↕' : '↔') : chip.icon }}</span>
            <span>{{ chip.label }}</span>
            <button @click="clearFilter(chip.type)" class="ml-0.5 text-gray-500 hover:text-white transition-colors cursor-pointer" aria-label="Remove filter">&times;</button>
          </span>
        </div>

        <!-- Row 2: Mode tabs + Categories + Featured Tags -->
        <div class="flex gap-1.5 mt-3 overflow-x-auto pb-1 scrollbar-thin -mx-4 sm:mx-0 px-4 sm:px-0 items-center" role="tablist" aria-label="Browse modes">
          <button @click="selectMode('random')" :class="mode==='random' && !selectedTag ? 'bg-indigo-600 text-white shadow-sm' : 'bg-gray-800/60 text-gray-400 hover:text-white hover:bg-gray-700/50'" class="shrink-0 px-3.5 py-1.5 rounded-full text-xs font-medium transition-all duration-150 min-h-[32px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/60" role="tab">
            <span class="flex items-center gap-1.5">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
              随机
            </span>
          </button>
          <button @click="selectMode('gallery')" :class="mode==='gallery' ? 'bg-indigo-600 text-white shadow-sm' : 'bg-gray-800/60 text-gray-400 hover:text-white hover:bg-gray-700/50'" class="shrink-0 px-3.5 py-1.5 rounded-full text-xs font-medium transition-all duration-150 min-h-[32px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/60" role="tab">
            <span class="flex items-center gap-1.5">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" /></svg>
              图集
            </span>
          </button>
          <span class="shrink-0 text-xs text-gray-500 font-medium mx-1 px-2 py-1 bg-gray-800/40 rounded-md">分类</span>
          <button v-for="cat in categories" :key="cat.name||cat" @click="selectCategory(cat)" :class="selectedCategory===(cat.name||cat) ? 'bg-indigo-600 text-white shadow-sm' : 'bg-gray-800/60 text-gray-400 hover:text-white hover:bg-gray-700/50'" class="shrink-0 px-3.5 py-1.5 rounded-full text-xs font-medium transition-all duration-150 min-h-[32px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/60" role="tab">{{ cat.name||cat }}</button>
          <span class="shrink-0 text-xs text-gray-500 font-medium mx-1 px-2 py-1 bg-gray-800/40 rounded-md">标签</span>
          <button v-for="tag in featuredTags" :key="tag.name||tag" @click="selectTag(tag)" :class="selectedTag===(tag.name||tag) ? 'bg-indigo-600 text-white shadow-sm' : 'bg-gray-800/60 text-gray-400 hover:text-white hover:bg-gray-700/50'" class="shrink-0 px-3.5 py-1.5 rounded-full text-xs font-medium transition-all duration-150 min-h-[32px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/60" role="tab">{{ tag.name||tag }}</button>
        </div>
      </div>
    </header>

    <!-- Main -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 py-6 sm:py-8">
      <!-- Loading skeleton -->
      <div v-if="loading && mode === 'random'" class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3 sm:gap-4">
        <div v-for="i in 6" :key="i" class="aspect-[3/4] rounded-xl bg-gray-800/60 animate-pulse" />
      </div>

      <!-- Error -->
      <div v-else-if="error" class="flex flex-col items-center justify-center py-24 text-gray-500 gap-4">
        <svg class="w-12 h-12 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
        <p class="text-sm">{{ error }}</p>
        <button @click="selectedTag?loadImages():loadRandom()" class="px-4 py-2 bg-gray-800 hover:bg-gray-700 rounded-lg text-sm text-gray-300 transition-colors duration-150 cursor-pointer">Retry</button>
      </div>

      <!-- Gallery: random gallery with its images -->
      <div v-else-if="mode === 'gallery'" class="space-y-4">
        <div class="flex items-center gap-2 flex-wrap">
          <svg class="w-5 h-5 text-indigo-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" /></svg>
          <span v-if="selectedGallery" class="text-sm text-gray-300 font-medium truncate flex-1 min-w-0">{{ selectedGallery.title || '随机图集' }}</span>
          <button @click="loadRandomGallery" class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-indigo-600/80 hover:bg-indigo-600 text-white rounded-lg text-xs font-medium transition-colors duration-150 cursor-pointer shrink-0" title="下一个随机图集">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
            下一个
          </button>
        </div>
        <!-- Loading skeleton -->
        <div v-if="loading" class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3 sm:gap-4">
          <div v-for="i in 6" :key="i" class="aspect-[3/4] rounded-xl bg-gray-800/60 animate-pulse" />
        </div>
        <!-- Gallery images -->
        <template v-else>
          <div v-if="images.length" class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3 sm:gap-4">
            <div v-for="img in images" :key="img.id" @click="openModal(img)" class="group relative cursor-pointer rounded-xl overflow-hidden bg-gray-800/40 aspect-[3/4] ring-1 ring-gray-800/50 hover:ring-indigo-500/30 transition-all duration-300">
              <img :src="img.url" alt="Gallery image" class="w-full h-full object-cover transition-all duration-500 group-hover:scale-[1.08] group-hover:brightness-110" loading="lazy" />
              <div class="absolute inset-0 bg-gradient-to-t from-black/40 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
            </div>
          </div>
          <div v-else-if="!loading" class="flex flex-col items-center justify-center py-24 text-gray-500 gap-4">
            <svg class="w-12 h-12 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
            <p class="text-sm">No images available</p>
          </div>
          <!-- Load More -->
          <div v-if="images.length && hasMore" class="flex flex-col items-center py-8 gap-4">
            <div id="scroll-sentinel" class="h-4" />
            <button v-if="!loadingMore" @click="loadMore" class="px-6 py-2.5 bg-gray-800 hover:bg-gray-700 text-gray-300 hover:text-white rounded-xl text-sm font-medium transition-all duration-150 cursor-pointer border border-gray-700/50">Load More</button>
            <div v-if="loadingMore" class="flex items-center gap-2 text-gray-500 text-sm">
              <div class="animate-spin h-4 w-4 border-2 border-indigo-500 border-t-transparent rounded-full" />
              Loading...
            </div>
          </div>
        </template>
      </div>

      <!-- Image grid (tag preview / random) -->
      <template v-else-if="mode !== 'gallery'">
        <div v-if="images.length" class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3 sm:gap-4">
          <div v-for="img in images" :key="img.id" @click="openModal(img)" class="group relative cursor-pointer rounded-xl overflow-hidden bg-gray-800/40 aspect-[3/4] ring-1 ring-gray-800/50 hover:ring-indigo-500/30 transition-all duration-300">
            <img :src="img.url" :alt="'Image from ' + (selectedTag || 'gallery')" class="w-full h-full object-cover transition-all duration-500 group-hover:scale-[1.08] group-hover:brightness-110" loading="lazy" />
            <div class="absolute inset-0 bg-gradient-to-t from-black/40 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
          </div>
        </div>
        <div v-else-if="!loading" class="flex flex-col items-center justify-center py-24 text-gray-500 gap-4">
          <svg class="w-12 h-12 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
          <p class="text-sm">Select a tag to browse images</p>
        </div>

        <!-- Scroll sentinel + Load More -->
        <div v-if="images.length && hasMore" class="flex flex-col items-center py-8 gap-4">
          <div id="scroll-sentinel" class="h-4" />
          <button v-if="!loadingMore" @click="loadMore" class="px-6 py-2.5 bg-gray-800 hover:bg-gray-700 text-gray-300 hover:text-white rounded-xl text-sm font-medium transition-all duration-150 cursor-pointer border border-gray-700/50">Load More</button>
          <div v-if="loadingMore" class="flex items-center gap-2 text-gray-500 text-sm">
            <div class="animate-spin h-4 w-4 border-2 border-indigo-500 border-t-transparent rounded-full" />
            Loading...
          </div>
        </div>
      </template>
    </main>

    <!-- Modal -->
    <Teleport to="body">
      <transition name="modal">
        <div v-if="showModal && modalImg" class="fixed inset-0 z-50 bg-black/95 flex flex-col md:flex-row" @touchmove.prevent role="dialog" aria-modal="true" aria-label="Image preview">
          <button @click="closeModal" class="fixed top-4 right-4 z-[60] w-10 h-10 flex items-center justify-center bg-black/50 hover:bg-gray-800/80 rounded-full text-white/70 hover:text-white transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-indigo-500/60" aria-label="Close preview">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
          <div class="flex-1 relative flex items-center justify-center overflow-hidden touch-none min-h-0 md:h-full" @wheel.prevent="handleWheel">
            <img :src="modalImg.url" alt="Preview image" class="max-w-full max-h-full object-contain transition-transform duration-200 ease-out select-none" :style="{ transform: 'scale('+zoom+') translate('+(panX/zoom)+'px,'+(panY/zoom)+'px)', cursor: zoom > 1 ? (isPanning ? 'grabbing' : 'grab') : 'pointer' }" draggable="false" @dblclick="handleDblClick" @touchstart="touchStartImg" @touchmove="touchMoveImg" @mousedown="panStart" @mousemove="panMove" @mouseup="panEnd" @mouseleave="panEnd" />
            <div v-if="zoom > 1" class="fixed bottom-6 left-1/2 -translate-x-1/2 bg-black/70 text-white/90 text-xs px-3.5 py-1.5 rounded-full font-medium tabular-nums backdrop-blur-sm">{{ (zoom * 100).toFixed(0) }}%</div>
            <div v-if="zoom === 1" class="fixed bottom-6 left-1/2 -translate-x-1/2 text-white/30 text-xs hidden sm:block">Ctrl+scroll or double-tap to zoom</div>
          </div>
          <div v-if="modalMeta" class="shrink-0 bg-gray-900/95 border-t md:border-t-0 md:border-l border-gray-800/60 p-5 text-sm space-y-4 md:w-72 md:h-full md:overflow-y-auto md:max-h-screen">
            <div><span class="text-gray-500 text-xs font-medium uppercase tracking-wider">ID</span><p class="text-white font-mono text-sm mt-0.5">{{ modalMeta.id }}</p></div>
            <div v-if="modalMeta.width"><span class="text-gray-500 text-xs font-medium uppercase tracking-wider">Dimensions</span><p class="text-white text-sm mt-0.5">{{ modalMeta.width }} &times; {{ modalMeta.height }}</p></div>
            <div v-if="modalMeta.orientation"><span class="text-gray-500 text-xs font-medium uppercase tracking-wider">Orientation</span><p class="text-white text-sm capitalize mt-0.5">{{ modalMeta.orientation }}</p></div>
            <div v-if="modalMeta.tags?.length"><span class="text-gray-500 text-xs font-medium uppercase tracking-wider">Tags</span><div class="flex flex-wrap gap-1.5 mt-1.5"><span v-for="tag in modalMeta.tags" :key="tag" class="px-2.5 py-1 bg-gray-800 text-gray-300 rounded-lg text-xs font-medium">{{ tag }}</span></div></div>
            <div v-if="modalMeta.gallery"><span class="text-gray-500 text-xs font-medium uppercase tracking-wider">Gallery</span><p class="text-white text-sm mt-0.5">{{ modalMeta.gallery.title || modalMeta.gallery.id }}</p></div>
          </div>
          <div v-else-if="modalImg.metaUrl" class="shrink-0 flex items-center justify-center p-8 md:w-72"><div class="animate-spin h-5 w-5 border-2 border-indigo-500 border-t-transparent rounded-full" /></div>
        </div>
      </transition>
    </Teleport>
    </template>
  </div>
</template>

<style scoped>
.modal-enter-active, .modal-leave-active { transition: opacity 0.2s ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
.scrollbar-thin { scrollbar-width: thin; scrollbar-color: #374151 transparent; }
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important; }
}
</style>
import Layout from "@/layout/index.vue";
import { createRouter, createWebHistory } from "vue-router";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/redirect",
      component: Layout,
      meta: {
        hidden: true,
      },
      children: [
        {
          path: ":path(.*)",
          component: () => import("@/views/redirect/index.vue"),
        },
      ],
    },
    {
      path: "/403",
      component: () => import("@/views/error/403.vue"),
      meta: {
        hidden: true,
      },
    },
    {
      path: "/tray-task",
      component: () => import("@/views/tray-task/index.vue"),
      meta: {
        hidden: true,
      },
    },
    {
      path: "/404",
      component: () => import("@/views/error/404.vue"),
      meta: {
        hidden: true,
      },
      alias: "/:pathMatch(.*)*",
    },
    {
      path: "/complete",
      component: Layout,
      redirect: "/complete",
      children: [
        {
          path: "complete",
          component: () => import("@/views/completetask/index.vue"),
          name: "complete",
          meta: {
            title: "已完成任务",
          },
        },
      ],
    },
    {
      path: "/unfinished",
      component: Layout,
      redirect: "/unfinished",
      children: [
        {
          path: "unfinished",
          component: () => import("@/views/alltask/index.vue"),
          name: "unfinished",
          meta: {
            title: "未完成",
          },
        },
      ],
    },

    {
      path: "/",
      component: Layout,
      redirect: "/run",
      children: [
        {
          path: "run",
          component: () => import("@/views/runtask/index.vue"),
          name: "run",
          meta: {
            title: "正在运行的任务",
          },
        },
      ],
    },
    {
      path: "/trash",
      component: Layout,
      redirect: "/trash",
      children: [
        {
          path: "trash",
          component: () => import("@/views/trashtask/index.vue"),
          name: "trash",
          meta: {
            title: "回收站",
          },
        },
      ],
    },
    {
      path: "/settings",
      component: Layout,
      redirect: "/settings",
      children: [
        {
          path: "settings",
          component: () => import("@/views/settings/index.vue"),
          name: "settings",
          meta: {
            title: "设置",
          },
        },
      ],
    },
  ],
});

export default router;

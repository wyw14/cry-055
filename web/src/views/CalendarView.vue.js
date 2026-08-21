import { computed, onMounted, ref } from 'vue';
import { ChevronLeft, ChevronRight } from 'lucide-vue-next';
import StatusBadge from '../components/StatusBadge.vue';
import { useInstrumentStore } from '../stores/instruments';
const store = useInstrumentStore();
const cursor = ref(new Date());
onMounted(store.load);
const title = computed(() => cursor.value.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long' }));
const days = computed(() => { const year = cursor.value.getFullYear(), month = cursor.value.getMonth(), first = new Date(year, month, 1), last = new Date(year, month + 1, 0), result = Array(first.getDay()).fill(null); for (let d = 1; d <= last.getDate(); d++)
    result.push(new Date(year, month, d)); return result; });
function shift(delta) { cursor.value = new Date(cursor.value.getFullYear(), cursor.value.getMonth() + delta, 1); }
function due(day) { if (!day)
    return []; return store.instruments.filter(item => { const value = new Date(item.next_due_at); return value.toDateString() === day.toDateString(); }); }
; /* PartiallyEnd: #3632/scriptSetup.vue */
function __VLS_template() {
    const __VLS_ctx = {};
    let __VLS_components;
    let __VLS_directives;
    __VLS_elementAsFunction(__VLS_intrinsicElements.section, __VLS_intrinsicElements.section)({});
    __VLS_elementAsFunction(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
        ...{ class: ("page-title") },
    });
    __VLS_elementAsFunction(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({});
    __VLS_elementAsFunction(__VLS_intrinsicElements.h1, __VLS_intrinsicElements.h1)({});
    __VLS_elementAsFunction(__VLS_intrinsicElements.p, __VLS_intrinsicElements.p)({});
    __VLS_elementAsFunction(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
        ...{ class: ("month-switch") },
    });
    __VLS_elementAsFunction(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
        ...{ onClick: (...[$event]) => {
                __VLS_ctx.shift(-1);
            } },
        ...{ class: ("icon") },
        title: ("上个月"),
    });
    const __VLS_0 = {}.ChevronLeft;
    /** @type { [typeof __VLS_components.ChevronLeft, ] } */ ;
    // @ts-ignore
    const __VLS_1 = __VLS_asFunctionalComponent(__VLS_0, new __VLS_0({}));
    const __VLS_2 = __VLS_1({}, ...__VLS_functionalComponentArgsRest(__VLS_1));
    __VLS_elementAsFunction(__VLS_intrinsicElements.strong, __VLS_intrinsicElements.strong)({});
    (__VLS_ctx.title);
    __VLS_elementAsFunction(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
        ...{ onClick: (...[$event]) => {
                __VLS_ctx.shift(1);
            } },
        ...{ class: ("icon") },
        title: ("下个月"),
    });
    const __VLS_6 = {}.ChevronRight;
    /** @type { [typeof __VLS_components.ChevronRight, ] } */ ;
    // @ts-ignore
    const __VLS_7 = __VLS_asFunctionalComponent(__VLS_6, new __VLS_6({}));
    const __VLS_8 = __VLS_7({}, ...__VLS_functionalComponentArgsRest(__VLS_7));
    __VLS_elementAsFunction(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
        ...{ class: ("calendar") },
    });
    for (const [label] of __VLS_getVForSourceType((['日', '一', '二', '三', '四', '五', '六']))) {
        __VLS_elementAsFunction(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
            key: ((label)),
            ...{ class: ("weekday") },
        });
        (label);
    }
    for (const [day, index] of __VLS_getVForSourceType((__VLS_ctx.days))) {
        __VLS_elementAsFunction(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
            key: ((index)),
            ...{ class: ("calendar-day") },
            ...{ class: (({ muted: !day })) },
        });
        if (day) {
            __VLS_elementAsFunction(__VLS_intrinsicElements.span, __VLS_intrinsicElements.span)({});
            (day.getDate());
        }
        for (const [item] of __VLS_getVForSourceType((__VLS_ctx.due(day)))) {
            __VLS_elementAsFunction(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
                key: ((item.id)),
                ...{ class: ("calendar-item") },
            });
            __VLS_elementAsFunction(__VLS_intrinsicElements.strong, __VLS_intrinsicElements.strong)({});
            (item.asset_number);
            // @ts-ignore
            /** @type { [typeof StatusBadge, ] } */ ;
            // @ts-ignore
            const __VLS_12 = __VLS_asFunctionalComponent(StatusBadge, new StatusBadge({
                status: ((item.status)),
            }));
            const __VLS_13 = __VLS_12({
                status: ((item.status)),
            }, ...__VLS_functionalComponentArgsRest(__VLS_12));
        }
    }
    ['page-title', 'month-switch', 'icon', 'icon', 'calendar', 'weekday', 'calendar-day', 'muted', 'calendar-item',];
    var __VLS_slots;
    var $slots;
    let __VLS_inheritedAttrs;
    var $attrs;
    const __VLS_refs = {};
    var $refs;
    var $el;
    return {
        attrs: {},
        slots: __VLS_slots,
        refs: $refs,
        rootEl: $el,
    };
}
;
const __VLS_self = (await import('vue')).defineComponent({
    setup() {
        return {
            ChevronLeft: ChevronLeft,
            ChevronRight: ChevronRight,
            StatusBadge: StatusBadge,
            title: title,
            days: days,
            shift: shift,
            due: due,
        };
    },
});
export default (await import('vue')).defineComponent({
    setup() {
        return {};
    },
    __typeEl: {},
});
; /* PartiallyEnd: #4569/main.vue */

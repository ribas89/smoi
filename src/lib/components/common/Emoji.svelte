<script>
	import { emojiFromCodePoints } from '$lib/utils/emojiUnicode';
	import { shortCodesToEmojis } from '$lib/stores';

	export let shortCode;
	export let className = 'size-4';
</script>

{#if $shortCodesToEmojis[shortCode]}
	<span
		class="inline-flex shrink-0 items-center justify-center align-middle leading-none {className}"
		role="img"
		aria-label={shortCode}
	>
		{emojiFromCodePoints($shortCodesToEmojis[shortCode])}
	</span>
{:else}
	<div>
		{shortCode}
	</div>
{/if}

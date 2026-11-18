/** Convert Unicode code-point sequences such as 1F44D-1F3FD to native emoji. */
export function emojiFromCodePoints(sequence: string): string {
    if (!/^[0-9a-f]+(?:-[0-9a-f]+)*$/i.test(sequence)) return sequence;

    try {
        return String.fromCodePoint(...sequence.split('-').map((part) => parseInt(part, 16)));
    } catch {
        return sequence;
    }
}

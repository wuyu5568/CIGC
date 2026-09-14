import i18n from "@/language";

const lang = (text: string, variables: any = {}) => {
    if (!text) return ''
    try {
        const words = i18n.global.t(text, variables);
        if (words == null || words === text) {
            return text;
        }
        return String(words);
    } catch (e) {
        return text;
    }
  }

export default lang
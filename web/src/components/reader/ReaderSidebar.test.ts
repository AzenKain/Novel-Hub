import { describe, expect, it } from "vitest";
import { getSidebarEntryKind } from "./ReaderSidebar";

describe("ReaderSidebar getSidebarEntryKind", () => {
  describe("Vietnamese language patterns", () => {
    it("classifies Vietnamese special chapters (prologues, epilogues, side stories, extras, cover, TOC)", () => {
      const specials = [
        "Chương cuối",
        "Chương cuối: Tạm biệt tất cả",
        "Chương cuối cùng",
        "Chương kết",
        "Chương kết: Hẹn gặp lại",
        "Hồi kết",
        "Hồi kết: Khép lại hành trình",
        "Kết đoạn",
        "Đoạn kết",
        "Phần kết",
        "Phần kết: Lời tạm biệt",
        "Phần mở đầu",
        "Kết thúc",
        "Kết cục",
        "Kết chương",
        "Vĩ thanh",
        "Vĩ khúc",
        "Chung chương",
        "Chung khúc",
        "Lời mở đầu",
        "Mở đầu",
        "Lời nói đầu",
        "Lời dẫn",
        "Lời tựa",
        "Tiền truyện",
        "Phi lộ",
        "Khởi đầu",
        "Dẫn nhập",
        "Lời bạt",
        "Hậu ký",
        "Lời kết",
        "Lời cảm ơn",
        "Lời tri ân",
        "Tổng kết",
        "Cuối sách",
        "Ngoại truyện",
        "Ngoại truyện 1: Một ngày bình thường",
        "[Ngoại truyện] Phần 1",
        "【Ngoại truyện】 Cuộc sống thường ngày",
        "Phiên ngoại: Chuyện tình yêu",
        "Phiên ngoại 1",
        "Phụ chương: Hồ sơ nhân vật",
        "Chương phụ 1",
        "Chương đặc biệt",
        "Ngoại chương",
        "Truyện phụ",
        "Khoảng nghỉ",
        "Chương xen",
        "Bìa sách",
        "Bìa",
        "Bìa trước",
        "Bìa sau",
        "Trang bìa",
        "Mục lục",
        "Minh họa",
        "Tranh minh họa",
        "Phụ lục",
        "Thông tin xuất bản",
        "Thông tin sách",
        "Giới thiệu nhân vật",
        "Hồ sơ nhân vật",
        "Bản quyền",
        "Ghi chú",
        "Chú thích",
      ];

      for (const title of specials) {
        expect(getSidebarEntryKind(title)).toBe("special");
      }
    });

    it("classifies Vietnamese section structural dividers", () => {
      const sections = [
        "Quyển 1: Thiếu niên xuất sơn",
        "Quyển thứ nhất",
        "Quyển I",
        "Tập 1: Vương giả trở lại",
        "Tập 2",
        "Tập thứ hai",
        "Phần 1: Mở đầu kỷ nguyên",
        "Phần thứ nhất",
        "Phần I",
        "Bộ 1: Thiên hạ vô song",
      ];

      for (const title of sections) {
        expect(getSidebarEntryKind(title)).toBe("section");
      }
    });

    it("classifies standard Vietnamese numbered chapters as chapters", () => {
      const chapters = [
        "Chương 1: Bắt đầu",
        "Chương 10: Vĩ thanh của kẻ phản bội",
        "Chương 5: Lời bạt của người lữ hành",
        "Hồi 1: Đào viên kết nghĩa",
        "Hồi thứ hai: Thảo lư tam cố",
        "Chương 999: Đại kết cục chi chiến",
      ];

      for (const title of chapters) {
        expect(getSidebarEntryKind(title)).toBe("chapter");
      }
    });
  });

  describe("English language patterns", () => {
    it("classifies English special entries", () => {
      const specials = [
        "Prologue",
        "Prologue: Awakening",
        "0. Prologue",
        "00 - Cover",
        "01 - Introduction",
        "Epilogue",
        "Epilogue: A New Dawn",
        "Afterword",
        "Author's Note",
        "Foreword",
        "Preface",
        "Introduction",
        "Side Story 1",
        "Side Story: The Past",
        "Extra: School Festival",
        "Extra Chapter",
        "Bonus Chapter",
        "Special Chapter",
        "Interlude: The King",
        "Intermission",
        "Spin-off: Side Adventures",
        "Omake",
        "Gaiden",
        "Short Story",
        "One-shot",
        "Cover",
        "Front Cover",
        "Back Cover",
        "Title Page",
        "Half Title",
        "Table of Contents",
        "Contents",
        "TOC",
        "Illustrations",
        "Color Pages",
        "Color Inserts",
        "Character Profiles",
        "Character Intro",
        "Dramatis Personae",
        "Glossary",
        "Appendix",
        "Final Chapter",
        "The End",
        "Acknowledgments",
        "Credits",
        "Copyright",
        "Colophon",
        "About the Author",
      ];

      for (const title of specials) {
        expect(getSidebarEntryKind(title)).toBe("special");
      }
    });

    it("classifies English section structural dividers", () => {
      const sections = [
        "Volume 1",
        "Vol. 1",
        "Vol 1: The Gathering",
        "Book 1: Heritage",
        "Book I",
        "Part 1: The Journey",
        "Part I",
        "Arc 1: Arrival",
        "Act I: The Call",
        "Saga of the North",
        "Season 1",
      ];

      for (const title of sections) {
        expect(getSidebarEntryKind(title)).toBe("section");
      }
    });

    it("classifies standard English numbered chapters", () => {
      const chapters = [
        "Chapter 1",
        "Chapter 1: The Beginning",
        "Chapter 50: The Final Confrontation",
        "Ch. 12 - Meeting",
        "The Dark Knight",
        "01 - The Beginning",
      ];

      for (const title of chapters) {
        expect(getSidebarEntryKind(title)).toBe("chapter");
      }
    });
  });

  describe("Chinese language patterns", () => {
    it("classifies Chinese special entries", () => {
      const specials = [
        "序章",
        "序言",
        "序幕",
        "楔子",
        "前言",
        "引子",
        "导言",
        "卷首",
        "开篇",
        "发端",
        "终章",
        "終章",
        "尾声",
        "尾聲",
        "后记",
        "後記",
        "结语",
        "结局",
        "完结",
        "大结局",
        "完结感言",
        "跋",
        "余话",
        "后话",
        "最终章",
        "最后一章",
        "谢幕",
        "番外",
        "番外篇：日常",
        "【番外】第一章",
        "外传",
        "幕间",
        "插曲",
        "特别篇",
        "小剧场",
        "附章",
        "短篇",
        "花絮",
        "插图",
        "彩页",
        "封面",
        "封底",
        "扉页",
        "目录",
        "人物介绍",
        "人物一览",
        "附录",
        "版权",
        "设定",
        "地图",
      ];

      for (const title of specials) {
        expect(getSidebarEntryKind(title)).toBe("special");
      }
    });

    it("classifies Chinese section structural dividers", () => {
      const sections = [
        "第一卷 崛起",
        "卷一",
        "分卷 1",
        "第一部 风云",
        "第一篇 启航",
        "第一集 降临",
        "第一册",
      ];

      for (const title of sections) {
        expect(getSidebarEntryKind(title)).toBe("section");
      }
    });

    it("classifies Chinese chapters", () => {
      const chapters = ["第一章 初入江湖", "第10章 风云突变", "100章 决战巅峰"];

      for (const title of chapters) {
        expect(getSidebarEntryKind(title)).toBe("chapter");
      }
    });
  });

  describe("Japanese language patterns", () => {
    it("classifies Japanese special entries", () => {
      const specials = [
        "プロローグ",
        "エピローグ",
        "序章",
        "序文",
        "はじめに",
        "終章",
        "あとがき",
        "後書き",
        "後記",
        "解説",
        "おわりに",
        "大団円",
        "番外編",
        "幕間",
        "外伝",
        "おまけ",
        "閑話",
        "間奏",
        "表紙",
        "裏表紙",
        "口絵",
        "挿絵",
        "カラーイラスト",
        "登場人物紹介",
        "目次",
        "付録",
        "奥付",
      ];

      for (const title of specials) {
        expect(getSidebarEntryKind(title)).toBe("special");
      }
    });

    it("classifies Japanese sections", () => {
      const sections = ["第1巻", "巻之一", "第1編", "第1部"];

      for (const title of sections) {
        expect(getSidebarEntryKind(title)).toBe("section");
      }
    });

    it("classifies Japanese chapters", () => {
      const chapters = ["第1話 はじまり", "第2章 旅立ち", "1話"];

      for (const title of chapters) {
        expect(getSidebarEntryKind(title)).toBe("chapter");
      }
    });
  });

  describe("Korean language patterns", () => {
    it("classifies Korean special entries", () => {
      const specials = [
        "프롤로그",
        "에필로그",
        "서문",
        "서장",
        "종장",
        "후기",
        "작가의 말",
        "맺음말",
        "외전 1",
        "[외전] 특별한 하루",
        "막간",
        "보너스",
        "표지",
        "일러스트",
        "삽화",
        "목차",
        "등장인물",
        "부록",
        "판권",
      ];

      for (const title of specials) {
        expect(getSidebarEntryKind(title)).toBe("special");
      }
    });

    it("classifies Korean sections", () => {
      const sections = ["제1권", "1권", "제1부", "1부"];

      for (const title of sections) {
        expect(getSidebarEntryKind(title)).toBe("section");
      }
    });

    it("classifies Korean chapters", () => {
      const chapters = ["1화 시작", "제1장 새로운 시작", "제1화"];

      for (const title of chapters) {
        expect(getSidebarEntryKind(title)).toBe("chapter");
      }
    });
  });

  describe("Multi-language global patterns (Spanish, French, German, Russian, Arabic, Hindi, Thai, Indonesian)", () => {
    it("classifies European language specials and sections", () => {
      expect(getSidebarEntryKind("Prólogo: La verdad")).toBe("special");
      expect(getSidebarEntryKind("Epílogo")).toBe("special");
      expect(getSidebarEntryKind("Capítulo final")).toBe("special");
      expect(getSidebarEntryKind("Portada")).toBe("special");
      expect(getSidebarEntryKind("Volumen 1")).toBe("section");
      expect(getSidebarEntryKind("Tomo 1")).toBe("section");
      expect(getSidebarEntryKind("Parte 1")).toBe("section");
      expect(getSidebarEntryKind("Avant-propos")).toBe("special");
      expect(getSidebarEntryKind("Épilogue")).toBe("special");
      expect(getSidebarEntryKind("Vorwort")).toBe("special");
      expect(getSidebarEntryKind("Nachwort")).toBe("special");
      expect(getSidebarEntryKind("Band 1")).toBe("section");
      expect(getSidebarEntryKind("Teil 1")).toBe("section");
      expect(getSidebarEntryKind("Пролог")).toBe("special");
      expect(getSidebarEntryKind("Эпилог")).toBe("special");
      expect(getSidebarEntryKind("Послесловие")).toBe("special");
      expect(getSidebarEntryKind("Обложка")).toBe("special");
      expect(getSidebarEntryKind("Том 1")).toBe("section");
      expect(getSidebarEntryKind("Часть 1")).toBe("section");
      expect(getSidebarEntryKind("Глава 1: Начало")).toBe("chapter");
      expect(getSidebarEntryKind("Capítulo 1: El despertar")).toBe("chapter");
      expect(getSidebarEntryKind("Chapitre 1: La rencontre")).toBe("chapter");
      expect(getSidebarEntryKind("Kapitel 1: Der Anfang")).toBe("chapter");
    });

    it("classifies Arabic, Hindi, Thai, Indonesian patterns", () => {
      expect(getSidebarEntryKind("مقدمة: بداية القصة")).toBe("special");
      expect(getSidebarEntryKind("خاتمة")).toBe("special");
      expect(getSidebarEntryKind("فصل إضافي")).toBe("special");
      expect(getSidebarEntryKind("مجلد 1")).toBe("section");
      expect(getSidebarEntryKind("جزء 1")).toBe("section");

      expect(getSidebarEntryKind("प्रस्तावना")).toBe("special");
      expect(getSidebarEntryKind("भूमिका")).toBe("special");
      expect(getSidebarEntryKind("उपसंहार")).toBe("special");
      expect(getSidebarEntryKind("अंतिम अध्याय")).toBe("special");
      expect(getSidebarEntryKind("विशेष अध्याय")).toBe("special");
      expect(getSidebarEntryKind("भाग 1")).toBe("section");
      expect(getSidebarEntryKind("खंड 1")).toBe("section");

      expect(getSidebarEntryKind("บทนำ")).toBe("special");
      expect(getSidebarEntryKind("คำนำ")).toBe("special");
      expect(getSidebarEntryKind("บทส่งท้าย")).toBe("special");
      expect(getSidebarEntryKind("ตอนพิเศษ")).toBe("special");
      expect(getSidebarEntryKind("ตอนจบ")).toBe("special");
      expect(getSidebarEntryKind("เล่ม 1")).toBe("section");
      expect(getSidebarEntryKind("ภาค 1")).toBe("section");

      expect(getSidebarEntryKind("Prakata")).toBe("special");
      expect(getSidebarEntryKind("Pengantar")).toBe("special");
      expect(getSidebarEntryKind("Bab Khusus")).toBe("special");
      expect(getSidebarEntryKind("Cerita Sampingan")).toBe("special");
      expect(getSidebarEntryKind("Epilog")).toBe("special");
      expect(getSidebarEntryKind("Jilid 1")).toBe("section");
      expect(getSidebarEntryKind("Bagian 1")).toBe("section");
    });
  });
});

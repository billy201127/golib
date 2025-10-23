package confuse

import (
	"testing"
)

func TestBasicUnitFunctionality(t *testing.T) {
	// 测试用例组
	tests := []struct {
		name     string
		seed     int
		input    string
		wantSame bool // 是否期望输出与输入相同
	}{
		{
			name:     "空字符串测试",
			seed:     12345,
			input:    "",
			wantSame: true,
		},
		{
			name:     "非词典词测试",
			seed:     12345,
			input:    "xyz123",
			wantSame: false, // 现在非词典词会使用字符级加密，所以会改变
		},
		{
			name:     "词典词测试",
			seed:     12345,
			input:    "algorithm", // 确保这个词在词典中
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建混淆器
			obf := NewObfuscatorSDK(tt.seed)

			// 执行混淆
			obfuscated := obf.ObfuscateWord(tt.input)

			// 执行反混淆
			deobfuscated := obf.DeobfuscateWord(obfuscated)

			// 验证可逆性
			if deobfuscated != tt.input {
				t.Errorf("可逆性测试失败: %s -> %s -> %s",
					tt.input, obfuscated, deobfuscated)
			}

			// 验证是否按预期改变
			if tt.wantSame && obfuscated != tt.input {
				t.Errorf("期望保持不变但发生改变: %s -> %s",
					tt.input, obfuscated)
			}
			if !tt.wantSame && obfuscated == tt.input {
				t.Errorf("期望发生改变但未改变: %s", tt.input)
			}

			// 打印映射结果（便于观察）
			t.Logf("映射结果: %s -> %s -> %s",
				tt.input, obfuscated, deobfuscated)
		})
	}
}

func TestModularInverse(t *testing.T) {
	tests := []struct {
		name string
		a    int
		m    int
		want int
	}{
		{
			name: "基本测试1",
			a:    3,
			m:    7,
			want: 5, // 因为 3 * 5 ≡ 1 (mod 7)
		},
		{
			name: "基本测试2",
			a:    5,
			m:    11,
			want: 9, // 因为 5 * 9 ≡ 1 (mod 11)
		},
		{
			name: "不存在逆元",
			a:    4,
			m:    8,
			want: -1, // 4和8不互质，不存在逆元
		},
		{
			name: "模数为1的特殊情况",
			a:    1,
			m:    1,
			want: 0, // 任何数mod 1都是0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modularInverse(tt.a, tt.m)
			if got != tt.want {
				t.Errorf("modularInverse(%d, %d) = %d, want %d",
					tt.a, tt.m, got, tt.want)
			}

			// 如果存在逆元且模数不为1，验证计算结果
			if tt.want != -1 && tt.m != 1 {
				result := (tt.a * got) % tt.m
				if result != 1 {
					t.Errorf("验证失败: %d * %d ≡ %d (mod %d), want 1",
						tt.a, got, result, tt.m)
				}
			}
		})
	}
}

func TestDictionaryConsistency(t *testing.T) {
	// 创建两个相同种子的混淆器
	seed := 12345
	obf1 := NewObfuscatorSDK(seed)
	obf2 := NewObfuscatorSDK(seed)

	// 测试词汇
	testWords := []string{
		"algorithm",
		"computer",
		"science",
		"mathematics",
	}

	for _, word := range testWords {
		// 使用两个混淆器分别混淆
		result1 := obf1.ObfuscateWord(word)
		result2 := obf2.ObfuscateWord(word)

		// 验证结果一致性
		if result1 != result2 {
			t.Errorf("相同种子产生不同结果: %s -> [%s, %s]",
				word, result1, result2)
		}

		// 验证可逆性
		deobf1 := obf1.DeobfuscateWord(result1)
		deobf2 := obf2.DeobfuscateWord(result2)

		if deobf1 != word || deobf2 != word {
			t.Errorf("反混淆结果不一致: %s -> %s -> [%s, %s]",
				word, result1, deobf1, deobf2)
		}

		t.Logf("一致性验证通过: %s -> %s -> %s",
			word, result1, deobf1)
	}
}

func TestSeedVariation(t *testing.T) {
	// 测试不同种子产生不同映射
	seeds := []int{1111, 2222, 3333, 4444, 5555}
	word := "algorithm"
	results := make(map[string]bool)

	t.Log("\n🔑 不同种子的映射测试")
	t.Log("===================")

	for _, seed := range seeds {
		obf := NewObfuscatorSDK(seed)
		result := obf.ObfuscateWord(word)

		// 验证结果不重复
		if results[result] {
			t.Errorf("种子 %d 产生重复的映射结果: %s", seed, result)
		}
		results[result] = true

		// 验证可逆性
		restored := obf.DeobfuscateWord(result)
		if restored != word {
			t.Errorf("种子 %d 的映射不可逆: %s -> %s -> %s",
				seed, word, result, restored)
		}

		t.Logf("种子 %d: %s → %s", seed, word, result)
	}
}

func TestEdgeCases(t *testing.T) {
	obf := NewObfuscatorSDK(12345)

	tests := []struct {
		name  string
		input string
	}{
		{"空字符串", ""},
		{"空格字符串", "   "},
		{"特殊字符", "!@#$%"},
		{"数字", "12345"},
		{"混合字符", "abc123!@#"},
		{"非ASCII字符", "你好世界"},
		{"超长字符串", "veryveryverylongwordthatprobablydoesnotexistinthedictionary"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试混淆
			obfuscated := obf.ObfuscateWord(tt.input)
			// 测试反混淆
			deobfuscated := obf.DeobfuscateWord(obfuscated)

			// 对于非词典词，应该保持不变
			if obfuscated != tt.input {
				t.Logf("非词典词发生改变: %s -> %s", tt.input, obfuscated)
			}

			// 验证可逆性
			if deobfuscated != tt.input {
				t.Errorf("边界情况可逆性失败: %s -> %s -> %s",
					tt.input, obfuscated, deobfuscated)
			}

			t.Logf("边界情况测试: %s -> %s -> %s",
				tt.input, obfuscated, deobfuscated)
		})
	}
}

func TestPerformance(t *testing.T) {
	obf := NewObfuscatorSDK(12345)
	word := "algorithm"

	// 混淆性能测试
	t.Run("混淆性能", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			result := obf.ObfuscateWord(word)
			if result == "" {
				t.Error("混淆失败")
			}
		}
	})

	// 反混淆性能测试
	t.Run("反混淆性能", func(t *testing.T) {
		obfuscated := obf.ObfuscateWord(word)
		for i := 0; i < 1000; i++ {
			result := obf.DeobfuscateWord(obfuscated)
			if result != word {
				t.Error("反混淆失败")
			}
		}
	})
}

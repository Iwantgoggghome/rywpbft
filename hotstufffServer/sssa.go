package main

import (
	"errors"
	"fmt"
	"math/big"
)

var (
	ErrCannotRequireMoreShares = errors.New("cannot require more shares then existing")
	ErrOneOfTheSharesIsInvalid = errors.New("one of the shares is invalid")
)

const (
	DefaultPrimeStr = "115792089237316195423570985008687907853269984665640564039457584007913129639747"
)

var MinT int              // 保存门限值
var TPK map[string]string // 保存公钥
var TSK map[string]string // 保存私钥
var strNodeName string    // 所用公钥和私钥的节点名

func init() {
	TPK = make(map[string]string) // 初始化所有的公钥数据都PKPool中
	TSK = make(map[string]string)
	strNodeName = fmt.Sprintf("N%d", nodeCount)
	pk := getPubKey(strNodeName)
	sk := getPivKey(strNodeName)
	TPK[strNodeName] = string(pk)
	TSK[strNodeName] = string(sk)
}

/**
 * Returns a new arary of secret shares (encoding x,y pairs as base64 strings)
 * created by Shamir's Secret Sharing Algorithm requring a minimum number of
 * share to recreate, of length shares, from the input secret raw as a string
 minimum指的是门限值
 shares是n的值，也就是成员数
 raw就是要分享的秘密
**/
func Create(minimum int, shares int, raw string) ([]string, error) {

	// Verify minimum isn't greater than shares; there is no way to recreate
	// the original polynomial in our current setup, therefore it doesn't make
	// sense to generate fewer shares than are needed to reconstruct the secret.
	MinT = minimum
	if minimum > shares {
		return []string{""}, ErrCannotRequireMoreShares
	}

	// Convert the secret to its respective 256-bit big.Int representation
	var secret []*big.Int = splitByteToInt([]byte(raw))

	// Set constant prime across the package
	prime, _ = big.NewInt(0).SetString(DefaultPrimeStr, 10)

	// List of currently used numbers in the polynomial
	var numbers []*big.Int = make([]*big.Int, 0)
	numbers = append(numbers, big.NewInt(0))

	// Create the polynomial of degree (minimum - 1); that is, the highest
	// order term is (minimum-1), though as there is a constant term with
	// order 0, there are (minimum) number of coefficients.
	//
	// However, the polynomial object is a 2d array, because we are constructing
	// a different polynomial for each part of the secret
	// polynomial[parts][minimum]
	var polynomial [][]*big.Int = make([][]*big.Int, len(secret))
	for i := range polynomial {
		polynomial[i] = make([]*big.Int, minimum)
		polynomial[i][0] = secret[i]

		for j := range polynomial[i][1:] {
			// Each coefficient should be unique
			number := random()
			for inNumbers(numbers, number) {
				number = random()
			}
			numbers = append(numbers, number)

			polynomial[i][j+1] = number
		}
	}

	// Create the secrets object; this holds the (x, y) points of each share.
	// Again, because secret is an array, each share could have multiple parts
	// over which we are computing Shamir's Algorithm. The last dimension is
	// always two, as it is storing an x, y pair of points.
	//
	// Note: this array is technically unnecessary due to creating result
	// in the inner loop. Can disappear later if desired. [TODO]
	//
	// secrets[shares][parts][2]
	var secrets [][][]*big.Int = make([][][]*big.Int, shares)
	var result []string = make([]string, shares)

	// For every share...
	for i := range secrets {
		secrets[i] = make([][]*big.Int, len(secret))
		// ...and every part of the secret...
		for j := range secrets[i] {
			secrets[i][j] = make([]*big.Int, 2)

			// ...generate a new x-coordinate...
			number := random()
			for inNumbers(numbers, number) {
				number = random()
			}
			numbers = append(numbers, number)

			// ...and evaluate the polynomial at that point...
			secrets[i][j][0] = number
			secrets[i][j][1] = evaluatePolynomial(polynomial[j], number)

			// ...add it to results...
			result[i] += toBase64(secrets[i][j][0])
			result[i] += toBase64(secrets[i][j][1])
		}
	}

	// ...and return!
	return result, nil
}

func VerifyThresholdSig(hashCode string, signInfo string) (bool, error) {
	fmt.Println(hashCode, signInfo)
	if TRsaVerySignWithSha256([]byte(hashCode), []byte(signInfo), []byte(TPK[strNodeName])) {
		return true, nil
	}
	return false, errors.New("VerifyThresholdSigchcuo")
}

// 传入shares是部分门限签名，如果到达了门限值，则对hashcode进行签名，以后好验证
func ThresholdSig(shares []string, hashCode string) (string, error) {
	var ncount int = 0
	for i := range shares {
		if IsValidShare(shares[i]) == true {
			ncount++
		}
	}
	if ncount < MinT {
		return "", errors.New("VerifyThresholdSigchcuo")
	} else {
		_, err := Combine(shares)
		if err != nil {
			fmt.Println("在sssa.go的ThresholdSig函数中出错了")
			return "", errors.New("VerifyThresholdSigchcuo")
		} else {
			signInfo := TRsaSignWithSha256(([]byte(hashCode)), []byte(TSK[strNodeName]))
			return string(signInfo), nil
		}

	}
}

/**
 * Takes a string array of shares encoded in base64 created via Shamir's
 * Algorithm; each string must be of equal length of a multiple of 88 characters
 * as a single 88 character share is a pair of 256-bit numbers (x, y).
 *
 * Note: the polynomial will converge if the specified minimum number of shares
 *       or more are passed to this function. Passing thus does not affect it
 *       Passing fewer however, simply means that the returned secret is wrong.
 如果传递进去大于等于门限值数目的partsig，则返回正确的结果，小于门限值，则返回错误的结果
 所以呢，这个是秘密共享的东西，不是门限签名的东西
**/
func Combine(shares []string) (string, error) {
	// Recreate the original object of x, y points, based upon number of shares
	// and size of each share (number of parts in the secret).
	var secrets [][][]*big.Int = make([][][]*big.Int, len(shares))

	// Set constant prime
	prime, _ = big.NewInt(0).SetString(DefaultPrimeStr, 10)

	// For each share...
	for i := range shares {
		// ...ensure that it is valid...
		if IsValidShare(shares[i]) == false {
			return "", ErrOneOfTheSharesIsInvalid
		}

		// ...find the number of parts it represents...
		share := shares[i]
		count := len(share) / 88
		secrets[i] = make([][]*big.Int, count)

		// ...and for each part, find the x,y pair...
		for j := range secrets[i] {
			cshare := share[j*88 : (j+1)*88]
			secrets[i][j] = make([]*big.Int, 2)
			// ...decoding from base 64.
			secrets[i][j][0] = fromBase64(cshare[0:44])
			secrets[i][j][1] = fromBase64(cshare[44:])
		}
	}

	// Use Lagrange Polynomial Interpolation (LPI) to reconstruct the secret.
	// For each part of the secert (clearest to iterate over)...
	var secret []*big.Int = make([]*big.Int, len(secrets[0]))
	for j := range secret {
		secret[j] = big.NewInt(0)
		// ...and every share...
		for i := range secrets { // LPI sum loop
			// ...remember the current x and y values...
			origin := secrets[i][j][0]
			originy := secrets[i][j][1]
			numerator := big.NewInt(1)   // LPI numerator
			denominator := big.NewInt(1) // LPI denominator
			// ...and for every other point...
			for k := range secrets { // LPI product loop
				if k != i {
					// ...combine them via half products...
					current := secrets[k][j][0]
					negative := big.NewInt(0)
					negative = negative.Mul(current, big.NewInt(-1))
					added := big.NewInt(0)
					added = added.Sub(origin, current)

					numerator = numerator.Mul(numerator, negative)
					numerator = numerator.Mod(numerator, prime)

					denominator = denominator.Mul(denominator, added)
					denominator = denominator.Mod(denominator, prime)
				}
			}

			// LPI product
			// ...multiply together the points (y)(numerator)(denominator)^-1...
			working := big.NewInt(0).Set(originy)
			working = working.Mul(working, numerator)
			working = working.Mul(working, modInverse(denominator))

			// LPI sum
			secret[j] = secret[j].Add(secret[j], working)
			secret[j] = secret[j].Mod(secret[j], prime)
		}
	}

	// ...and return the result!
	return string(mergeIntToByte(secret)), nil
}

/**
 * Takes in a given string to check if it is a valid secret
 *
 * Requirements:
 * 	Length multiple of 88
 *	Can decode each 44 character block as base64
 *
 * Returns only success/failure (bool)
**/
func IsValidShare(candidate string) bool {
	// Set constant prime across the package
	prime, _ = big.NewInt(0).SetString(DefaultPrimeStr, 10)

	if len(candidate)%88 != 0 {
		return false
	}

	count := len(candidate) / 44
	for j := 0; j < count; j++ {
		part := candidate[j*44 : (j+1)*44]
		decode := fromBase64(part)
		if decode.Cmp(big.NewInt(0)) == -1 || decode.Cmp(prime) == 1 {
			return false
		}
	}

	return true
}

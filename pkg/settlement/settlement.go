package settlement

import (
	"expense-tracker-api/internal/models"
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Balance represents a user's net balance (positive = owed to them, negative = they owe)
type Balance struct {
	UserID   uuid.UUID
	UserName string
	Amount   decimal.Decimal
}

// Transaction represents a single settlement transaction
type Transaction struct {
	FromUserID   uuid.UUID
	FromUserName string
	ToUserID     uuid.UUID
	ToUserName   string
	Amount       decimal.Decimal
}

// Calculator handles debt settlement calculations
type Calculator struct {
	precision int32
}

// NewCalculator creates a new settlement calculator
func NewCalculator() *Calculator {
	return &Calculator{
		precision: 4, // 4 decimal places for currency precision
	}
}

// CalculateSimplifiedDebts calculates the minimum number of transactions needed
// to settle all debts using an optimized greedy algorithm.
//
// Algorithm Explanation:
// 1. Calculate net balance for each user (what they're owed minus what they owe)
// 2. Separate users into creditors (positive balance) and debtors (negative balance)
// 3. Sort both lists by absolute amount (largest first) for optimal matching
// 4. Greedily match debtors with creditors to minimize transactions
//
// Time Complexity: O(n log n) due to sorting
// Space Complexity: O(n)
func (c *Calculator) CalculateSimplifiedDebts(balances []Balance) []Transaction {
	if len(balances) == 0 {
		return []Transaction{}
	}

	// Separate creditors (positive) and debtors (negative)
	var creditors, debtors []Balance

	for _, b := range balances {
		// Round to precision and ignore very small amounts
		amount := b.Amount.Round(c.precision)
		if amount.Abs().LessThan(decimal.NewFromFloat(0.01)) {
			continue // Skip negligible amounts
		}

		if amount.GreaterThan(decimal.Zero) {
			creditors = append(creditors, Balance{UserID: b.UserID, UserName: b.UserName, Amount: amount})
		} else if amount.LessThan(decimal.Zero) {
			debtors = append(debtors, Balance{UserID: b.UserID, UserName: b.UserName, Amount: amount.Abs()})
		}
	}

	// Sort by amount descending (largest first) for optimal matching
	sort.Slice(creditors, func(i, j int) bool {
		return creditors[i].Amount.GreaterThan(creditors[j].Amount)
	})
	sort.Slice(debtors, func(i, j int) bool {
		return debtors[i].Amount.GreaterThan(debtors[j].Amount)
	})

	var transactions []Transaction
	creditorIdx, debtorIdx := 0, 0

	// Greedily match debtors with creditors
	for creditorIdx < len(creditors) && debtorIdx < len(debtors) {
		creditor := &creditors[creditorIdx]
		debtor := &debtors[debtorIdx]

		// Determine the settlement amount (minimum of the two)
		settlementAmount := decimal.Min(creditor.Amount, debtor.Amount)

		// Create transaction
		transactions = append(transactions, Transaction{
			FromUserID:   debtor.UserID,
			FromUserName: debtor.UserName,
			ToUserID:     creditor.UserID,
			ToUserName:   creditor.UserName,
			Amount:       settlementAmount.Round(c.precision),
		})

		// Update balances
		creditor.Amount = creditor.Amount.Sub(settlementAmount)
		debtor.Amount = debtor.Amount.Sub(settlementAmount)

		// Move to next creditor/debtor if settled
		if creditor.Amount.LessThanOrEqual(decimal.NewFromFloat(0.01)) {
			creditorIdx++
		}
		if debtor.Amount.LessThanOrEqual(decimal.NewFromFloat(0.01)) {
			debtorIdx++
		}
	}

	return transactions
}

// CalculateGroupBalances calculates the net balance for each user in a group
// based on expenses and settlements.
func (c *Calculator) CalculateGroupBalances(
	userIDs []uuid.UUID,
	userNames map[uuid.UUID]string,
	expenses []models.Expense,
	settlements []models.Settlement,
) []Balance {
	balances := make(map[uuid.UUID]decimal.Decimal)

	// Initialize all users with zero balance
	for _, userID := range userIDs {
		balances[userID] = decimal.Zero
	}

	// Process expenses
	for _, expense := range expenses {
		// Person who paid gets credit
		if current, exists := balances[expense.PaidByID]; exists {
			balances[expense.PaidByID] = current.Add(expense.Amount)
		}

		// Each share holder gets debit
		for _, share := range expense.Shares {
			if current, exists := balances[share.UserID]; exists {
				balances[share.UserID] = current.Sub(share.Amount)
			}
		}
	}

	// Process settlements (they reduce debts)
	for _, settlement := range settlements {
		if settlement.Status != models.SettlementStatusCompleted {
			continue
		}

		// Person who paid (from) gets credit (their debt is reduced)
		if current, exists := balances[settlement.FromUserID]; exists {
			balances[settlement.FromUserID] = current.Add(settlement.Amount)
		}

		// Person who received (to) gets debit (they're owed less now)
		if current, exists := balances[settlement.ToUserID]; exists {
			balances[settlement.ToUserID] = current.Sub(settlement.Amount)
		}
	}

	// Convert to slice
	result := make([]Balance, 0, len(balances))
	for userID, amount := range balances {
		name := userNames[userID]
		if name == "" {
			name = "Unknown"
		}
		result = append(result, Balance{
			UserID:   userID,
			UserName: name,
			Amount:   amount.Round(c.precision),
		})
	}

	return result
}

// ValidateSplit validates that expense shares sum up to the total amount
func (c *Calculator) ValidateSplit(totalAmount decimal.Decimal, shares []decimal.Decimal, splitType models.SplitType) (bool, decimal.Decimal) {
	switch splitType {
	case models.SplitTypeEqual:
		// For equal split, any number of shares is valid
		return true, decimal.Zero

	case models.SplitTypeExact:
		// For exact split, shares must sum to total
		sum := decimal.Zero
		for _, share := range shares {
			sum = sum.Add(share)
		}
		difference := totalAmount.Sub(sum).Abs()
		// Allow small rounding differences
		return difference.LessThan(decimal.NewFromFloat(0.01)), difference

	case models.SplitTypePercent:
		// For percent split, percentages must sum to 100
		sum := decimal.Zero
		for _, share := range shares {
			sum = sum.Add(share)
		}
		difference := decimal.NewFromInt(100).Sub(sum).Abs()
		return difference.LessThan(decimal.NewFromFloat(0.01)), difference

	default:
		return false, decimal.Zero
	}
}

// CalculateEqualShares calculates equal shares for a given total and number of people
func (c *Calculator) CalculateEqualShares(totalAmount decimal.Decimal, numPeople int) []decimal.Decimal {
	if numPeople <= 0 {
		return []decimal.Decimal{}
	}

	baseShare := totalAmount.Div(decimal.NewFromInt(int64(numPeople)))
	shares := make([]decimal.Decimal, numPeople)

	// Round to 2 decimal places for currency
	roundedShare := baseShare.Round(2)

	// Calculate the difference due to rounding
	totalRounded := roundedShare.Mul(decimal.NewFromInt(int64(numPeople)))
	difference := totalAmount.Sub(totalRounded)

	// Distribute the difference to the first person
	for i := 0; i < numPeople; i++ {
		if i == 0 {
			shares[i] = roundedShare.Add(difference).Round(2)
		} else {
			shares[i] = roundedShare
		}
	}

	return shares
}

// CalculatePercentShares calculates shares based on percentages
func (c *Calculator) CalculatePercentShares(totalAmount decimal.Decimal, percentages []decimal.Decimal) []decimal.Decimal {
	shares := make([]decimal.Decimal, len(percentages))

	for i, percent := range percentages {
		share := totalAmount.Mul(percent).Div(decimal.NewFromInt(100))
		shares[i] = share.Round(2)
	}

	// Adjust for rounding differences
	totalShares := decimal.Zero
	for _, share := range shares {
		totalShares = totalShares.Add(share)
	}
	difference := totalAmount.Sub(totalShares)

	// Add difference to the first share if significant
	if difference.Abs().GreaterThanOrEqual(decimal.NewFromFloat(0.01)) && len(shares) > 0 {
		shares[0] = shares[0].Add(difference).Round(2)
	}

	return shares
}

// GetTransactionCount returns the number of transactions needed without settlements
func (c *Calculator) GetTransactionCountWithoutSettlement(balances []Balance) int {
	count := 0
	for _, b := range balances {
		if b.Amount.Abs().GreaterThanOrEqual(decimal.NewFromFloat(0.01)) {
			count++
		}
	}
	return count
}

// GetOptimizationStats returns statistics about the optimization
func (c *Calculator) GetOptimizationStats(originalDebts int, optimizedTransactions int) map[string]interface{} {
	reduction := 0
	if originalDebts > 0 {
		reduction = ((originalDebts - optimizedTransactions) * 100) / originalDebts
	}

	return map[string]interface{}{
		"original_debts":         originalDebts,
		"optimized_transactions": optimizedTransactions,
		"reduction_percentage":   reduction,
		"transactions_saved":     originalDebts - optimizedTransactions,
	}
}

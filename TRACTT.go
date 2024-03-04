package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/signintech/gopdf"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Transaction struct {
	Timestamp time.Time
	Type      string
	Amount    float64
}

type FinanceTracker struct {
	Transactions []Transaction
}

func (ft *FinanceTracker) AddTransaction(transactionType string, amount float64) {
	timestamp := time.Now()
	transaction := Transaction{Timestamp: timestamp, Type: transactionType, Amount: amount}
	ft.Transactions = append(ft.Transactions, transaction)
}

func (ft *FinanceTracker) CalculateTotals() (totalDebt float64, totalCredit float64) {
	for _, transaction := range ft.Transactions {
		if transaction.Type == "Credit" {
			totalCredit += transaction.Amount
		} else if transaction.Type == "Debt" {
			totalDebt += transaction.Amount
		}
	}
	return totalDebt, totalCredit
}

func (ft *FinanceTracker) GetTransactionsLast30Days() (last30DaysTransactions []Transaction, totalSpentLast30Days float64) {
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	for _, transaction := range ft.Transactions {
		if transaction.Timestamp.After(thirtyDaysAgo) {
			last30DaysTransactions = append(last30DaysTransactions, transaction)
			totalSpentLast30Days += transaction.Amount
		}
	}
	return last30DaysTransactions, totalSpentLast30Days
}

func generatePDFToBytes(transactions []Transaction, totalSpent float64, totalDebt float64, totalCredit float64) ([]byte, error) {
	var pdfBuffer bytes.Buffer

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: gopdf.Rect{W: 595.28, H: 841.89}}) // A4 size
	pdf.AddPage()
	err := pdf.AddTTFFont("Arial", "C:/Windows/Fonts/arial.ttf")
	if err != nil {
		return nil, err
	}
	err = pdf.SetFont("Arial", "", 14)
	if err != nil {
		return nil, err
	}

	// Add header
	pdf.Cell(nil, "Transactions in the last 30 days:")
	pdf.Br(20)

	// Add transaction details
	for _, transaction := range transactions {
		pdf.Cell(nil, fmt.Sprintf("Timestamp: %s, Type: %s, Amount: %.2f", transaction.Timestamp.Format("2006-01-02 15:04:05"), transaction.Type, transaction.Amount))
		pdf.Br(20)
	}

	// Add total spent
	pdf.Br(20)
	pdf.Cell(nil, fmt.Sprintf("Total spent in the last 30 days: %.2f", totalSpent))
	pdf.Br(20)

	// Add total debt
	pdf.Cell(nil, fmt.Sprintf("Total debt: %.2f", totalDebt))
	pdf.Br(20)

	// Add total credit
	pdf.Cell(nil, fmt.Sprintf("Total credit: %.2f", totalCredit))
	pdf.Br(20)

	// Save PDF to buffer
	err = pdf.Write(&pdfBuffer)
	if err != nil {
		return nil, err
	}

	return pdfBuffer.Bytes(), nil
}

func main() {
	tracker := FinanceTracker{}

	b, err := tb.NewBot(tb.Settings{
		Token:  "Bot Token",
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		fmt.Println("Error creating bot:", err)
		return
	}

	b.Handle("/start", func(m *tb.Message) {
		b.Send(m.Sender, "Welcome to the Finance Tracker Bot!")
	})

	b.Handle("/exit", func(m *tb.Message) {
		b.Send(m.Sender, "Exiting Finance Tracker Bot.")
		b.Stop()
	})

	b.Handle("/Last30", func(m *tb.Message) {
		last30Transactions, totalSpentLast30Days := tracker.GetTransactionsLast30Days()
		var response strings.Builder
		response.WriteString("Transactions in the last 30 days:\n")
		for _, transaction := range last30Transactions {
			response.WriteString(fmt.Sprintf("Timestamp: %s, Type: %s, Amount: %.2f\n", transaction.Timestamp.Format("2006-01-02 15:04:05"), transaction.Type, transaction.Amount))
		}
		response.WriteString(fmt.Sprintf("Total spent in the last 30 days: %.2f\n", totalSpentLast30Days))

		totalDebt, totalCredit := tracker.CalculateTotals()
		response.WriteString(fmt.Sprintf("Total Debt: %.2f\n", totalDebt))
		response.WriteString(fmt.Sprintf("Total Credit: %.2f\n", totalCredit))

		// Generate PDF
		pdfBytes, err := generatePDFToBytes(last30Transactions, totalSpentLast30Days, totalDebt, totalCredit)
		if err != nil {
			response.WriteString(fmt.Sprintf("Error generating PDF: %v\n", err))
		} else {
			response.WriteString("PDF generated successfully.")

			// Send PDF as a document to the user
			document := &tb.Document{File: tb.FromReader(bytes.NewReader(pdfBytes)), FileName: "finance_report.pdf"}
			b.Send(m.Sender, document)
		}
		b.Send(m.Sender, response.String())
	})

	b.Handle(tb.OnText, func(m *tb.Message) {
		input := m.Text

		parts := strings.Split(input, " ")
		if len(parts) != 2 {
			b.Send(m.Sender, "Invalid input. Please enter transaction in the format '/Type Amount'.")
			return
		}

		transactionType := parts[0]
		amount, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			b.Send(m.Sender, "Invalid input. Amount must be a number.")
			return
		}

		if transactionType != "/Debt" && transactionType != "/Credit" {
			b.Send(m.Sender, "Invalid transaction type. Please use '/Debt' or '/Credit'.")
			return
		}

		tracker.AddTransaction(transactionType[1:], amount)
		b.Send(m.Sender, "Transaction added successfully.")
	})

	fmt.Println("Bot is running...")
	b.Start()
}

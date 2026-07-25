package handlers

import (
	"encoding/json"
	"net/http"
	"opims/database"
	"opims/models"
	"strconv"
	"strings"
)

// SubcontractorsList handles GET (list) / POST (create) for subcontractor library.
func (h *Handler) SubcontractorsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		q := r.URL.Query()
		where := "WHERE 1=1"
		args := []interface{}{}

		for _, f := range []struct{ param, col string }{
			{"type", "registration_type"},
			{"country", "country"},
			{"category", "profession_category"},
			{"profession", "profession"},
			{"parent", "parent_short_name"},
		} {
			if v := q.Get(f.param); v != "" {
				where += " AND " + f.col + "=?"
				args = append(args, v)
			}
		}
		if kw := q.Get("keyword"); kw != "" {
			where += " AND (short_name LIKE ? OR full_name LIKE ?)"
			like := "%" + kw + "%"
			args = append(args, like, like)
		}

		rows, err := database.DB.Query(
			`SELECT sb.id, sb.short_name, sb.full_name, sb.registration_type, sb.country,
			        sb.profession_category, sb.profession, sb.other_professions,
			        sb.parent_short_name, sb.legal_rep_name, sb.legal_rep_id, sb.legal_rep_phone,
			        sb.contact_name, sb.contact_title, sb.contact_phone, sb.contact_email,
			        sb.biz_license, sb.tax_id, sb.reg_address, sb.reg_capital, sb.notes,
			        COALESCE((SELECT COUNT(*) FROM subcontractor_projects WHERE sub_short_name=sb.short_name), 0) as project_count,
			        COALESCE((SELECT SUM(contract_amount) FROM subcontractor_projects WHERE sub_short_name=sb.short_name), 0) as total_amount
			 FROM subcontractors_base sb `+where+" ORDER BY sb.id DESC", args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type row struct {
			models.SubcontractorBase
			ProjectCount float64 `json:"project_count"`
			TotalAmount  float64 `json:"total_amount"`
		}
		var list []row
		for rows.Next() {
			var r row
			rows.Scan(&r.ID, &r.ShortName, &r.FullName, &r.RegistrationType, &r.Country,
				&r.ProfessionCategory, &r.Profession, &r.OtherProfessions,
				&r.ParentShortName, &r.LegalRepName, &r.LegalRepID, &r.LegalRepPhone,
				&r.ContactName, &r.ContactTitle, &r.ContactPhone, &r.ContactEmail,
				&r.BizLicense, &r.TaxID, &r.RegAddress, &r.RegCapital, &r.Notes,
				&r.ProjectCount, &r.TotalAmount)
			list = append(list, r)
		}
		json.NewEncoder(w).Encode(list)

	case "POST":
		var b models.SubcontractorBase
		json.NewDecoder(r.Body).Decode(&b)
		_, err := database.DB.Exec(
			`INSERT INTO subcontractors_base (short_name,full_name,registration_type,country,
			 profession_category,profession,other_professions,parent_short_name,
			 legal_rep_name,legal_rep_id,legal_rep_phone,
			 contact_name,contact_title,contact_phone,contact_email,
			 biz_license,tax_id,reg_address,reg_capital,notes)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			b.ShortName, b.FullName, b.RegistrationType, b.Country,
			b.ProfessionCategory, b.Profession, b.OtherProfessions, b.ParentShortName,
			b.LegalRepName, b.LegalRepID, b.LegalRepPhone,
			b.ContactName, b.ContactTitle, b.ContactPhone, b.ContactEmail,
			b.BizLicense, b.TaxID, b.RegAddress, b.RegCapital, b.Notes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "created"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SubcontractorsByID handles GET / PUT / DELETE for a single subcontractor.
func (h *Handler) SubcontractorsByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/api/subcontractors/"))

	switch r.Method {
	case "GET":
		var b models.SubcontractorBase
		err := database.DB.QueryRow(
			`SELECT id, short_name, full_name, registration_type, country,
			        profession_category, profession, other_professions, parent_short_name,
			        legal_rep_name, legal_rep_id, legal_rep_phone,
			        contact_name, contact_title, contact_phone, contact_email,
			        biz_license, tax_id, reg_address, reg_capital, notes
			 FROM subcontractors_base WHERE id=?`, id,
		).Scan(&b.ID, &b.ShortName, &b.FullName, &b.RegistrationType, &b.Country,
			&b.ProfessionCategory, &b.Profession, &b.OtherProfessions, &b.ParentShortName,
			&b.LegalRepName, &b.LegalRepID, &b.LegalRepPhone,
			&b.ContactName, &b.ContactTitle, &b.ContactPhone, &b.ContactEmail,
			&b.BizLicense, &b.TaxID, &b.RegAddress, &b.RegCapital, &b.Notes)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(b)

	case "PUT":
		var b models.SubcontractorBase
		json.NewDecoder(r.Body).Decode(&b)
		_, err := database.DB.Exec(
			`UPDATE subcontractors_base SET short_name=?,full_name=?,registration_type=?,country=?,
			 profession_category=?,profession=?,other_professions=?,parent_short_name=?,
			 legal_rep_name=?,legal_rep_id=?,legal_rep_phone=?,
			 contact_name=?,contact_title=?,contact_phone=?,contact_email=?,
			 biz_license=?,tax_id=?,reg_address=?,reg_capital=?,notes=?,
			 updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			b.ShortName, b.FullName, b.RegistrationType, b.Country,
			b.ProfessionCategory, b.Profession, b.OtherProfessions, b.ParentShortName,
			b.LegalRepName, b.LegalRepID, b.LegalRepPhone,
			b.ContactName, b.ContactTitle, b.ContactPhone, b.ContactEmail,
			b.BizLicense, b.TaxID, b.RegAddress, b.RegCapital, b.Notes, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "updated"})

	case "DELETE":
		database.DB.Exec("DELETE FROM subcontractors_base WHERE id=?", id)
		json.NewEncoder(w).Encode(map[string]string{"ok": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
